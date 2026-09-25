# j2go

## Status / next steps (round 7)

**Round 6: done, verified, green. A real macOS window with a working button opens from Go,
`CGO_ENABLED=0`.** `mvn -q -f tooling/j2go/pom.xml package && bash tooling/port.sh &&
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...` all pass (same 33 tests).
`cmd/hello` (hand-written) builds a Display, a Shell with FillLayout, a PUSH Button "Hello" with a
Selection listener, `pack`/`open`, and runs the `readAndDispatch`/`sleep` loop. Verified: the
window appears (titlebar + "Hello" button, 82x56 after `pack`), `performClick:` on the button
prints the listener's line through the real `sendSelection -> postEvent -> runDeferredEvents`
path, `shell.close()` disposes and `display.dispose()` returns. `screencapture` has no Screen
Recording permission on this machine, so the check used an in-process
`cacheDisplayInRect:toBitmapImageRep:` snapshot (a scratch program, not in the repo).

**Translated for real this round** (all in `port.sh`'s swt invocation): `Display` (6861 lines),
`Device`, `DeviceData`, `Resource`, `Font`, `FontData`, `Color`, `Synchronizer`, `RunnableLock`,
`Monitor`, `TouchSource`. Their old stubs are gone (`widgets_monitor_manual.go` kept empty).
Still stubs (not on the Shell+Button path): `GC`, `Image`, `Cursor`, `Menu`/`MenuItem` (Display's
app menu is built from raw `NSMenu`, not SWT's `Menu`), `Caret`, `IME`, `ScrollBar`, `ToolBar`,
`Region`, `Accessible`, plus new opaque ones in `swt/widgets_stubs3_manual.go` (dialogs, tray,
taskbar, combo, `FontMetrics`, `LONG`, `Display.APPEARANCE`, `DPIUtil`/`BidiUtil`/`ImageUtil`/
`Compatibility`/`DefaultExceptionHandler` statics).

**Contract changes** are listed in "Round 6" at the end: functional values, anonymous classes,
`synchronized`, `java.util` containers, exceptions as `error`, the Callback bridge, by-pointer
native struct params, nil-safe casts/upcasts/instanceof, and more.

**Markers left** (swt invocation; cocoa has none besides the 166 known field/method name
clashes): MethodInvocation 54, ClassInstanceCreation 6, CatchClause 5, instanceof 4,
ExpressionMethodReference 2, MethodDeclaration 2. All are JDK surface off this path:
`StringBuilder`/`String.indexOf(s, from)`/`Integer.parseInt` (FontData string form), Resource's
leak tracker (`Cleaner`, `AtomicBoolean`, `ThreadGroup`), `Runtime.Version.parse` (AWT check,
short-circuited), `Synchronizer.moveAllEventsTo`'s `list::add` refs, `Thread.sleep`, the two
generic methods (`Widget.getTypedListeners`, `Display.syncCall`).

**Next round, in order:**
1. **Paint and custom drawing**: `GC` (+ `GCData` for real), `Image`, `Region`, `Path`,
   `Pattern`, `Transform`, `TextLayout`. `Control.drawWidget` short-circuits today only because no
   PaintListener is registered.
2. **More widgets**: `Label`, `Text`, `Composite` children, `Menu`/`MenuItem`, `ScrollBar`,
   `GridLayout`/`FormLayout`. Each widget class registered in `Display.initClasses` already has
   its ObjC subclass and callbacks - only the Java class is missing.
3. **NSException**: purego can't catch one. SWT's os.c returns 0 from a throwing native; only
   `NSColor.colorSpace` is guarded so far (`internal/cocoa/nsexception_manual.go`). Any other
   throwing selector aborts the process - find them as widgets arrive.
4. **Real Java enums** (`Display.APPEARANCE` is hand-written) and Java generics
   (`getTypedListeners`, `syncCall`).
5. `Shell` window placement: `pack()` gives a correct size; the window opened at x=0, y=1014
   (Cocoa coordinates) - not compared against real SWT's `cascadeWindow` result yet.

Java -> Go source translator for porting Eclipse SWT to Go. v0: hand-tuned to the graphics
value classes (`Point`, `Rectangle`, `RGB`, `RGBA`). v0.1 adds a second Go package,
`internal/cocoa`, for the `Eclipse SWT PI/cocoa` bindings, plus `native` method support. Built
on the Eclipse JDT DOM `ASTParser` with binding resolution.

`maven-shade-plugin`'s `createDependencyReducedPom` is off (`pom.xml`) - it's a build artifact
with no reason to exist in the repo (was previously gitignored but regenerated on every build).

## Source layout

`dev.gowt.j2go`: `Main` (CLI, AST parsing, file output), `TypeModel` (class hierarchy/Go names),
`Names` (capitalization/overload naming), `GoTypes` (Java type -> Go type text), `Manual`
(names for hand-written *_manual.go members), `Natives` (C symbol overrides), `Selectors`
(selector-string table).

`dev.gowt.j2go.emit`: the AST -> Go text emitter, split by responsibility so a change to one
construct touches one file. `Emitter` is the facade - shared per-compilation-unit state (model,
names, current class, prelude buffer, temp-var counter, escape/label stack) plus thin
cross-component delegators (`expr`/`stmt`/`block`/...); components call each other only through
it, never directly. Where to add a new construct:

- **new JDK method mapping** (`Math.min`, `String.equals`, ...) -> `JdkIntrinsics`
- **new statement form** -> `StatementEmitter`
- **new Java construct in a class body** (field, method shell) -> `ClassEmitter`
- **constructors, instance initializers, `super(...)`/`this(...)`/`super.method()`** ->
  `ConstructorEmitter` (split out of `ClassEmitter` in Round 5 - line budget)
- **new native-method shape** (purego binding, `_sizeof`, `_stret`, constant accessor) ->
  `NativeEmitter`
- **switch/throw/try-catch-finally/synchronized** -> `ControlFlowEmitter`
- **new expression form** (literal, prefix/postfix, assignment, name/field resolution, string
  escaping, switch expressions) -> `ExpressionEmitter`
- **lambdas, method references, anonymous classes** -> `FunctionalEmitter` (Round 6)
- **infix operators, numeric widening/adaptation, object upcasting, zero values** ->
  `NumericEmitter` (split out of `ExpressionEmitter` in Round 5 - line budget)
- **method invocation / `new` / argument adaptation** -> `InvocationEmitter`
- **instanceof, cast, or the impl-cascade helpers** -> `TypeTestEmitter`
- **try/catch escape detection** -> `EscapeScanner` (the ASTVisitor pre-scan)
- **a stateless text/binding helper shared by several components** -> `EmitUtil`
- **cross-package qualification (`<pkg>.` + import, package layering guard)** -> `PackageQualifier`
  (split out of `Emitter` in Round 10 - line budget)
- **the reflect registry (package `swt/swtreflect`)** -> `ReflectEmitter` (Round 11)
- **JUnit: `Assertions`/`Assumptions` calls, the per-class test registry** -> `TestEmitter` (Round 12)

## Run

```sh
tooling/port.sh
```

Builds the translator (`mvn -q -f tooling/j2go/pom.xml package`), runs it over the stage-1 and
stage-2 file lists, writes `swt/graphics_{point,rectangle,rgb,rgba}.go` and
`internal/cocoa/cocoa_*.go`, then `gofmt -w`s them.

Direct CLI:

```sh
mvn -q -f tooling/j2go/pom.xml package
java -jar tooling/j2go/target/j2go.jar --swt <swt repo root> --out . \
	org/eclipse/swt/graphics/Point.java org/eclipse/swt/graphics/Rectangle.java ...
```

`--out` is the repo root, not a single package directory: each file lands under `swt/`,
`internal/cocoa/` or `examples/<name>/` per its own Java package (`GoTypes.goPackageDir`; see
"Multiple Go packages" and "Round 10 controlexample" below).

File arguments are resolved against the SWT source roots (`Eclipse SWT/common`, `/cocoa`,
`Eclipse SWT PI/common`, `/cocoa` - whichever exist under `--swt`), or taken as-is if absolute.
`ASTParser` is set up with all of them as sourcepath, `includeRunningVMBootclasspath=true` (so
`java.lang.*`/`java.io.*` resolve off the running JDK 21), compliance 21, and bindings +
bindings recovery + statements recovery on. `createASTs` parses the whole requested-file list
in one pass so cross-file bindings (e.g. Rectangle referencing Point) resolve.

## Translation rules as implemented

Follows the contract in the task brief. Notes on the parts that needed a concrete decision the
contract left open:

- **One Go package `swt`**, file `<javapackage-last-segment>_<class lowercase>.go`. Header is
  `// Code generated by j2go from <path> @ af630a9093. DO NOT EDIT.` followed by the original
  license comment, copied verbatim from the source text before the `package` keyword.
- **Types**: `int/long/short/byte/char/float/double/boolean/String/Object` map as specified.
  Class types map to `*GoType`, except a manual type explicitly marked as a value type (see
  RoundingMode below).
- **Numeric conversions**: implicit Java widening is made explicit wherever two operand/target
  Go types differ (binary operands, assignment/return/argument/array-element targets), skipping
  literal operands (Go's untyped constants already convert). Binary operand promotion picks the
  wider of the two Go types (int8 < int16/uint16 < int32 < int64 < float32 < float64), so e.g.
  Java's `float - int` becomes `hue - float32(i)`, and `float32Expr + <double literal>` becomes
  `float64(float32Expr) + 0.5` (matching Java's real double promotion when the literal has no
  `f` suffix).
- **Overload naming**: first-declared keeps the base name; others get base name plus
  capitalized parameter names, in order. Contract gap: a later overload that has zero
  parameters (Rectangle's private no-arg constructor) has nothing to append and would collide
  with the base name, so it gets the parameter count appended instead (`initRectangle0`).
- **Inheritance / impl dispatch**: root struct gets an unexported `impl <Root>Impl` field
  whenever the hierarchy has subclasses in the translated set; `<Root>Impl` lists methods
  overridden anywhere in the hierarchy, typed with the *root's own* declared signature (Go has
  no covariant return types, so `Point.OfFloat.clone()`'s `Point.OfFloat` return becomes `*Point`
  at the interface level). A call to an overridden method always goes through `x.impl.M(...)`.
  When the call site's Java-resolved (covariant) return type is narrower than the interface's
  root type, the result is downcast back with the same cascade helper used for `instanceof`
  (see below) - this is what makes `Point.OfFloat.from`/`Rectangle.OfFloat.from` (which call
  `.clone()` on a variable statically typed as the *non-root* `OfFloat`, but must stay correct
  if the runtime object is the further-derived `WithMonitor`) type-check and dispatch correctly.
  A plain (non-overridden) method's own return type, if it still doesn't match the enclosing
  method's declared return type (e.g. a static factory returning the root type after building a
  subclass), is upcast with `&tmp.<TargetGoType>` (a promoted embedded-field address; Go
  promotes multi-level embedding, so this works regardless of how deep the target is).
- **instanceof / pattern matching**: `expr instanceof T name` and `expr instanceof T` (no
  pattern) both lower to `name, ok := <root>ImplAs<T>(x)`, hoisted as a prelude statement before
  the enclosing `if`/ternary; `ok` (or `_, ok`) replaces the instanceof expression inline.
  `x` is `expr.impl` when `expr`'s Java static type is itself one of our translated classes
  (its dynamic subtype identity lives in `.impl`), or `expr` directly when the static type is
  `java.lang.Object` (an `Equals(object any)` parameter already holds the concrete pointer).
  `<root>ImplAs<T>` is generated once per (root, target) pair actually used, as a type switch
  over every concrete class in the translated set that is `T` or a descendant of `T`, each case
  returning the pointer (or `&v.<T>` for a proper descendant, again via promoted-field upcast).
  Contract gap: a non-leaf instanceof target (e.g. `Point.OfFloat`, which `Point.WithMonitor`
  extends) cannot be a single Go type assertion, since a `*Point_WithMonitor` boxed value is not
  assertable to `*Point_OfFloat` even though it "is-a" one in Java; the generated switch is what
  makes this correct.
- **Ternaries**: Go has no conditional expression. Every `cond ? a : b` (as a var-decl
  initializer, a plain assignment RHS, or nested inside an argument/return expression) is
  lowered uniformly to a temp var declared with the ternary's resolved Go type, plus an
  `if/else` assigning it; nested lowering composes through the same "prelude" mechanism used for
  instanceof. Not attempted: a bare ternary as a method's whole return expression (not present
  in the 4 files) - would need the same treatment but wasn't exercised.
- **`switch` expression with `yield`**: only the shape actually used by
  `RGB(float,float,float)` is supported - `target = switch (e) { case C -> { stmts; yield v; }
  ... };` lowers to a `switch` statement where each arm's statements are emitted as-is and the
  `yield` becomes `target = v`. Any other switch-expression shape hits the unsupported marker.
- **Chained assignment** (`r = g = b = brightness;`) flattens right-to-left into sequential
  statements reusing each intermediate target as the next source, preserving Java's
  single-evaluation-of-the-rightmost-expression semantics.
- **String concatenation** (`"Point {" + x + ", " + y + "}"`): not in the contract. Any `+`
  chain whose resolved type is `java.lang.String` is flattened into one `fmt.Sprintf`, with a
  `%d/%v/%s/%t` verb chosen from each non-literal operand's Go type.
- **`java.lang.Math`**: not in the contract either. `Math.max`/`Math.min` (the only overload the
  4 files use, `(float,float)`) map inline to `math.Max`/`math.Min` with explicit
  `float64`/`float32` conversions. No other `Math` members are handled.
- **Manual/out-of-set types** (`tooling/j2go/manual.txt`, hand-written in `swt/*_manual.go`):
  - `org.eclipse.swt.SWT` - whole class skipped (5000+ lines, out of v0 scope). Stub provides
    `Error(code int32)` (panics) and the two constants the 4 files reference. Static references
    use the class's real naming rule (no-prefix exception for SWT, e.g. `SWT.error` -> `Error`,
    still capitalized - only the class-name prefix is dropped).
  - `org.eclipse.swt.widgets.Monitor` - opaque stub struct; only ever carried as a pointer, no
    method of it is called by the 4 files.
  - `org.eclipse.swt.graphics.RoundingMode` - the contract has no rule for Java enums. Hand
    translated as a Go value type (`type RoundingMode int32` + `Round(float32) int32`), marked
    as a value type in `GoTypes`/`Manual` so field/parameter types don't get an incorrect `*`.
    Its zero value is `ROUND` to match the Java field defaults j2go drops (see next point).
- **Field initializers** (`private RoundingMode locationRounding = RoundingMode.ROUND;` in
  `Rectangle.OfFloat`): implemented per JLS 8.8.7/12.5. Each `init<N>` that does not start with
  `this(...)` gets the enclosing class's non-static field initializers and instance-initializer
  blocks (`{ ... }` at class scope) inlined right after the super call (explicit
  `this.<Super>.init...(...)`, or the implicit zero-arg one, or nothing if there is none), in
  source declaration order, interleaving fields and blocks as they appear. A `this(...)`
  delegate skips them - JLS runs them exactly once, in whichever non-delegating constructor is
  ultimately reached. Static field initializers already had `var`/`const` support
  (`emitStaticFields`); static `{ ... }` blocks (`static { Library.loadLibrary(...); }` in
  `OS.java`) had no code path at all and were silently dropped - now each becomes its own
  `func init()`, emitted right after the static fields. Go runs all package-level `var`
  initializers (dependency-ordered) before any `func init()`, and same-file `func init()`s run
  in declaration order, so this matches JLS ordering except for the (currently unseen) case of a
  static block whose side effect must interleave with a *specific* var's initializer expression -
  not handled; would need converting that var to an assignment inside `func init()` too.
- **Exceptions**: `throw`/try/catch would hit the unsupported marker (none of the 4 files use
  them). `SWT.error(...)`'s manual stub panics, matching the "throw -> panic" rule.
- **Java keyword-shaped param/local names** (`objc_msgSend`'s cousin
  `CALLBACK_cellBaselineOffset(long func)` in `OS.java`): a Java identifier that happens to be a
  Go keyword (`func`, `type`, `map`, `range`, ...) gets a trailing `_` wherever it's declared or
  referenced (`paramList`, `argNames`, `emitVarDecl`, `emitSimpleName`) - `sanitizeIdent`.
- **`char` literals**: not in the contract. `'m'` (as in `('m' << 24) + ...`) emits verbatim -
  JDT's `CharacterLiteral.getEscapedValue()` is already valid Go rune-literal syntax for the
  plain-ASCII literals SWT uses.
- **Unsupported-marker closures now carry the right return type**: the immediately-invoked
  `func() any { panic(...) }()` fallback (unresolved call/new, unsupported prefix op, catch-all)
  is exactly what the "Unsupported constructs" section below already documented, but the code
  always used `any` regardless of context - fine as long as nothing assigned the result to a
  concretely-typed target, which none of the 4 stage-1 files ever did. `OS.java` immediately did
  (`int sizeof = OS.NSPoint_sizeof();` with `OS` unresolved): `any` doesn't assign to `int32`, so
  `gofmt`/`go build` failed downstream. Fixed once, generically: `panicClosure(e, message)` types
  the closure to `e.resolveTypeBinding()` (falling back to `any` only if that's itself
  unresolvable) and every call site (catch-all, `SwitchExpression`, `PrefixExpression`,
  unresolved `MethodInvocation`/`ClassInstanceCreation`) now goes through it.

## Multiple Go packages (`internal/cocoa`)

`org.eclipse.swt.internal.cocoa` translates into its own Go package, `internal/cocoa` (Go name
`cocoa`), instead of `swt`. Mechanically this only changes `Main`: each `CompilationUnit`'s own
Java package picks its output directory and `package` clause (`goPackageDir`/`goPackageName`);
`Emitter`/`TypeModel`/`GoTypes` are unaware of packages and were not touched for this - every
class handed to one `j2go` invocation is still assumed mutually referenceable without
qualification. That holds for stage 2 because every `internal/cocoa` file translated so far only
references other `internal/cocoa` classes; it will stop holding the moment a `swt`-package class
is translated alongside a `cocoa`-package one in the same run (Step 3), which needs real
cross-package qualification (`cocoa.NSPoint` + an import) in `GoTypes.map` - not implemented, not
needed by anything translated so far. Package-name-based cycle check ("`internal/cocoa` must not
import `swt`") is likewise not implemented as code; it holds today by construction (the PI layer
has nothing to import from `swt`) and needs enforcing once cross-package refs exist.

## Struct (value-type) classes

`NSPoint`, `NSRect`, `CGPoint`, `objc_super`, ... are plain data: a Java class with no explicit
superclass and no methods besides (optionally) `toString`. `TypeModel.ClassInfo.isStruct` flags
these during `collect()`; `GoTypes.map` emits the bare type name for them instead of `*T`, and
`emitNew`/`zeroValue` treat `new X()` / an absent field initializer as the zero-value composite
literal `X{}` rather than a generated constructor call - these classes have no declared
constructor at all (Java's implicit no-arg one only), so `emitConstructor` never runs for them,
and there is nothing to call. Not handled: a struct class whose implicit constructor would need
to run a *non-zero* field initializer (none exist in the translated set - `CGRect`'s
`origin = new CGPoint()` happens to equal `CGPoint{}` already).

## Native methods

`Modifier.isNative` methods (`emitStaticNativeMethod`; only static natives exist in
`OS.java`, `513/513`) resolve in this order:
1. **`<Struct>_sizeof()`** (zero args, name ends `_sizeof`, `<Struct>` a translated `isStruct`
   class): needs no native call at all - `unsafe.Sizeof(<Struct>{})` is exact, since the Go
   struct's field layout is a faithful copy of the Java `@field` layout.
2. **`tooling/j2go/natives.properties`** (`javaMethodBaseName=<libExpr>,<symbol>`, loaded by
   `Natives`): emits `var goName func(params) ret` plus a
   `purego.RegisterLibFunc(&goName, <libExpr>, "<symbol>")` line, collected per file into one
   `func init()` (natives first, then `deferredStaticInits` - see below). `<libExpr>` is a Go
   expression text, e.g. `libobjc()` (a lazy-`sync.Once` accessor in
   `internal/cocoa/libs_manual.go`, not a plain var - see "purego verification" for why).
3. Otherwise: unsupported-marker stub, `func goName(params) ret { panic("j2go: unsupported
   native <name>") }` - same shape as any other unsupported construct, just as a full
   declaration instead of an expression closure (natives have no body to embed one in).

Only 9 of `OS.java`'s ~20 `objc_msgSend*` shapes are mapped (see natives.properties for the
arm64-ABI reasoning); `objc_msgSend_stret`/`objc_msgSendSuper*` (24 methods) are deliberately
left as unsupported-marker stubs, not guessed.

**Method/field name collision** (found translating `OS.java`, not fixed): Java lets a class
declare a field and a method with the same name (`public static final int VERSION;` and
`public static int VERSION(int,int,int)`; `NSString kUTTypeFileURL = new
NSString(kUTTypeFileURL())` - field and native method both literally named `kUTTypeFileURL`).
Static fields and static methods both name to `<goFuncPrefix><Capitalize(javaName)>` with no
collision check (`emitStructFields` only checks instance-field-vs-instance-method, via
`methodGoNames`/the `FieldMethodNameClash` marker); for statics this produces two conflicting
`OSVERSION` declarations - `go build` catches it as `redeclared in this block`, but j2go itself
doesn't detect or avoid it. This pattern repeats roughly 100+ times in `OS.java` (every
`NSString`-constant field shadows the native that fills it) and blocks translating it as-is;
needs a static-namespace collision check mirroring the existing instance one before `OS.java` is
attempted again.

## Deferred static initialization (`func init()`)

Go initializes every package-level `var` (in dependency order) before any `func init()` runs, so
a static field initializer that either needs prelude statements (a ternary or `instanceof` inside
it - nowhere to put them in a bare `var X = expr` line) or calls anything at all can't be a `var`
initializer: a native isn't wired up (its `func`-typed var isn't assigned) until `func init()`
calls `purego.RegisterLibFunc`, and `OS.java`'s `sel_x`/`class_x` constants
(`sel_registerName("alloc")`-shaped, ~1360 of them) are exactly this. `emitStaticFields` now emits
such a field as a bare `var X T` plus a `X = expr` line (with its own prelude) collected into
`Emitter.deferredStaticInits`, folded into the same `func init()` as `nativeRegistrations` -
natives first, so a deferred field's own native calls are already wired up by the time its
assignment runs (within one `func init()`, ordinary sequential-statement order, not dependent on
Go's cross-file `func init()` ordering at all). The same "final + primitive/String -> `const`"
rule now excludes any initializer containing a call, since a call is never a Go constant
expression either (previously would have produced `const X = someFunc()`, invalid Go - not
reachable from the 4 stage-1 files, first hit by `OS.java`).

## Callback design (`internal/cocoa/callback_manual.go`)

SWT's `Callback(object, "method", argCount)` binds a Java method by reflection and hands out a
C trampoline (`callback.c`) whose all-`long` args/return make it usable as an ObjC IMP. Since
Round 6 the translator resolves the method at translate time instead (`InvocationEmitter.
emitCallback`): the target class is the `TypeLiteral`'s, else the enclosing class (`this`,
`getClass()`, a `Class<?>` local), the method is found by name + arg count, and

```go
DisplayWindowCallback3 = NewCallbackFn(func(args []int64) int64 { return DisplayWindowProcIdSelArg0(args[0], args[1], args[2]) }, 3)
```

`NewCallbackFn`/`Callback` are hand-written in `swt/widgets_stubs3_manual.go`;
`cocoa.NewCallbackN(argCount, fn)` builds the `purego.NewCallback` with `argCount` `uintptr`
args and remembers `fn` by its C address. `OS.CALLBACK_x(proc)` (os.c's by-value-struct
trampolines) is generated from the native's own `@method callback_types=...,callback_flags=...`
Javadoc (`NativeEmitter.emitStructCallback`): a Go IMP with the real ObjC shape (`NSRect`/
`NSPoint`/`NSSize`/`NSRange` by value, struct returns by value), which looks `proc` up and calls
it with each struct's address - pinned with `runtime.Pinner` so a stack move can't invalidate
it - and, for a struct return, copies and `C.free`s the pointer `windowProc` `C.malloc`ed, like
os.c does. `isFlipped_CALLBACK` (a fixed "YES" IMP) and `OS.call(proc, id, sel)` are hand-written
next to it. purego callbacks are never freed (`Callback.dispose` is a no-op); SWT creates ~20 of
the 2000 available, `CALLBACK_*` ~35 more.

## purego verification (darwin/arm64)

Before writing any of the above, a 20-line smoke test (not part of the module; the two programs
live in the task's scratch directory, not committed) confirmed, against real system libraries
(`libobjc.A.dylib`, `AppKit`), that `purego v0.11.1` on this machine (darwin/arm64, Go 1.27):
- handles plain `objc_msgSend` (id/SEL args, `bool` return) correctly;
- handles **struct-by-value both as an `objc_msgSend` argument and as its direct return** (no
  `_stret` involved) - built an `NSView`, sent `initWithFrame:` with an `NSRect` value arg, read
  it back via `frame` with an `NSRect`-typed return, byte-for-byte round trip. arm64's objc
  runtime has one entry point regardless of return shape (unlike i386/x86_64, no `_stret`
  variant exists), so this is the *only* correct way to bind a struct-returning message send -
  confirmed by there being no such call in `OS.java`'s own `objc_msgSend(...)` overloads (all
  return `long`; struct returns are exclusively `objc_msgSend_stret`'s explicit-out-param, void-
  return shape, which does not match arm64's ABI - see "Native methods" above);
- registering the same C symbol name (`"objc_msgSend"`) multiple times via
  `purego.RegisterLibFunc` against *different* Go function-pointer types works - each call site
  gets its own correctly-typed trampoline, which is exactly what SWT's per-overload native
  declarations need;
- `class_addMethod` + a `purego.NewCallback`-backed IMP: created a dynamic `NSObject` subclass at
  runtime, added a method whose implementation is a Go closure, called it through `objc_msgSend`,
  got the closure's return value back correctly.

No `_stret` case, no `objc_msgSendSuper`, no >2000-argument-shape stress test was run - those stay
unverified (see "Native methods"). purego's callback pool is capped at `maxCB = 2000`
(`syscall_unix.go`, applies to darwin); the cocoa PI layer needs ~25 (one per multiplexed
`Callback`, see above), comfortably inside it. CGO_ENABLED=0 throughout; no cgo was used or
needed.

## Unsupported constructs

Anything not covered above emits, at the point of use:

```go
panic("j2go: unsupported <NodeType>") // TODO(gowt-port): <original java, one line>
```

(as an expression, wrapped in an immediately-invoked closure of the right return type) and is
collected into the end-of-run summary, grouped by node kind. Zero markers were needed for the
4 stage-1 files, and zero for the 11 stage-2 struct files (`tooling/port.sh`'s shipped set).

**`OS.java` translation attempt** (superseded, see "Round 2" below): the first attempt (513
natives + ~1500 fields alongside only the 11 struct files) produced a large summary of
`ClassInstanceCreation`/`MethodInvocation`/`NativeMethod`/`StaticField` markers and did not
compile - `OS.java` pulls in `NSString extends NSObject extends id` and a nested `Selector`
enum, none translated yet. The one bug it surfaced and fixed at the time: `GoTypes.map`'s
`"unsupported_type_" + qualified` fallback used the dotted Java qualified name verbatim, which
`gofmt` parses as a chain of selector expressions - a real parse error. Now
`qualified.replace('.', '_')`: still undefined (an unsupported marker, as intended), but a
valid Go identifier.

## Round 2: direction changes 1-5, OS.java + id/NSObject/NSString cascade (checkpoint a)

`tooling/port.sh` now runs stage 2 (structs) and stage 3 (`C`, `Protocol`, `id`, `NSObject`,
`NSString`, `NSProcessInfo`, `OS`) as **one** `j2go` invocation, not two: `TypeModel` is
per-invocation, and `OS.java`/`NSString.java` construct stage-2 struct types (`new
NSOperatingSystemVersion()`, ...) directly - split across two JVM runs those come back
"unresolved type". `CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...` all pass with
this checkpoint's file set; `go test` actually exercises the natives (package `init()` really
calls into libobjc/AppKit), not just compiles them.

### Direction 1+2: generic native binding, no more natives.properties as the main path

`Natives`/`emitStaticNativeMethod` no longer look up a fixed `(libVar, symbol)` pair per native
base name. Every static native now gets:

```go
var <goName>_impl func(<params>) <ret>
var <goName>_once sync.Once
func <goName>(<params>) <ret> {
	<goName>_once.Do(func() { ensureFrameworks(); purego.RegisterLibFunc(&<goName>_impl, purego.RTLD_DEFAULT, "<symbol>") })
	return <goName>_impl(<args>)
}
```

`<symbol>` is `natives.properties`'s override for the Java base name if one exists, else the
literal Java method name - no Javadoc `accessor=`/`flags=` parsing is implemented (zero
occurrences of `accessor=` exist anywhere in `id`/`NSObject`/`NSString`/`OS`/`C`/`Protocol`, so
this was YAGNI'd; a file that needs it will need this added, see "Known gaps" below).
`purego.RTLD_DEFAULT` searches every already-`Dlopen`'d image, so no per-native library handle is
threaded through anymore; `internal/cocoa/libs_manual.go`'s `ensureFrameworks()` (`sync.Once`)
`Dlopen`s `libobjc`, `Foundation`, `AppKit`, `CoreFoundation`, `CoreGraphics`, `CoreText`,
`QuartzCore` once. **Not** a plain package `func init()`: Go runs same-package `func init()`s in
the compiler's (lexical filename) order, and a generated file's own deferred-static-field `func
init()` (`cocoa_os.go`, alphabetically before `libs_manual.go`) would then run *before* anything
was `Dlopen`'d - `class_x` fields would call `objc_getClass` against nothing loaded yet and
silently come back 0 (this was an actual bug here, caught by the diagnostic in "Round 2 bugs"
below, not hypothetical). Every native's own lazy-binding closure calls `ensureFrameworks()`
first, so it's independent of file init order regardless of which native runs first. A
missing/wrong symbol panics inside `RegisterLibFunc` on
that native's own first call, not at package init, exactly as directed - except for a native
called from a static field's own deferred initializer (next section), which is unavoidably as
eager as the field itself.

`_stret`/`_stretSuper` (`emitStretNative`): detected structurally (name ends `_stret`, return
type `void`, first param maps to a translated struct/`isStruct` class) - no per-method
allowlist. The real symbol is the Java name with `_stret` stripped (`objc_msgSend_stret` ->
`"objc_msgSend"`, `objc_msgSendSuper_stret` -> `"objc_msgSendSuper"`), bound the same lazy way
but typed to *return* the struct; `result` becomes `*Struct` in the Go signature and the call
writes `*result = <backing>(...)`. Callers (`emitMethodInvocation`) take `&` of their own
struct-valued argument for exactly this native shape. 24/24 objc_msgSend(Super)_stret overloads
in `OS.java` now translate and are exercised (`NSObject.draggingLocation`,
`NSProcessInfo.operatingSystemVersion`, both under test).

`C.PTR_sizeof()`: SWT's own native (`sizeof(void*)`), not a system-framework symbol - special
cased next to the existing `<Struct>_sizeof()` case, `int32(unsafe.Sizeof(uintptr(0)))`.

### Direction 5: static field/method name collision

Resolved the same direction the existing instance-level `FieldMethodNameClash` rule already
picked: the **method keeps the base name, the field gets a `_` suffix**
(`staticFieldClashesWithMethod` checks `ITypeBinding.getDeclaredMethods()` for a same-named
static method; applied consistently at the field's own declaration and at every reference site,
`staticFieldRef`). 166 real occurrences in `OS.java` (`VERSION`, `kUTTypeFileURL`, ~160
`NSAccessibility*`/`NSPrint*`/attribute-name constants each shadowing the native that fills
them) - all resolve to `<Field>_` with zero remaining collisions. Bug fixed alongside this:
`emitSimpleName` unconditionally prefixed `this.` for *any* field reference, including an
unqualified reference to the enclosing class's own **static** field (`VERSION = VERSION(...)`
inside `OS.java`'s own `static {}` block) - now routes through the same `staticFieldRef` static
fields use everywhere else.

### Selector enum elision

`Selector` (`org.eclipse.swt.internal.cocoa.Selector`, a 1360-constant enum, one
`sel_registerName(name)` call per constant in its private constructor) is not translated - Java
enum declarations/enum-constant-with-constructor-args are a new AST shape the contract has no
rule for, and this enum's only real job, `OS.sel_registerName(name)`, is trivially inlinable.
`registerSelector`/`getSelector`/`SELECTORS` (`java.util.Map<Long,Selector>`, OS.java's own
bookkeeping for a reverse value->Selector lookup) are unused anywhere outside `OS.java`/
`Selector.java` themselves (checked by grep across the whole SWT tree) and are dropped entirely
(`manual.txt`, `Manual.SKIP_METHOD_NAMES`/`isSkippedField`).

Every one of `OS.java`'s ~1360 `sel_x`/`class_x` fields reads `Selector.sel_x.value` (field name
== enum constant name, always). `tooling/j2go/selectors.properties` (regex-extracted once from
`Selector.java`'s `name("arg")` constant declarations, not hand-maintained) maps
`enumConstName -> selector string`; `Emitter.selectorStringOf`/`selectorValueRewrite` recognize
the `<EnumType>.<const>.value` shape structurally (qualifier resolves to an enum constant
binding) and rewrite it directly to `OSSel_registerName("<string>")` - no enum AST support
needed at all. `containsCall` (decides const vs. deferred-`var` for a static field) was extended
to recognize this shape too, since it isn't a `MethodInvocation` in the Java AST but does need
to defer to `func init()` like one.

### NSObject dependency stubs

`NSObject.java`'s own methods reference 11 cocoa types outside this checkpoint's translated set
purely as parameter/return types, never calling a method on them (`.id`/`.Id` only): `NSImage`,
`NSWindow`, `NSPasteboard`, `NSOutlineView`, `NSTableColumn`, `NSCell`, `NSTableView`, `NSArray`,
`NSURLAuthenticationChallenge`, `NSURLCredential`, `DOMEvent`. Each is a 1-line opaque struct in
`internal/cocoa/stubs_manual.go` embedding the real translated `id` (same idea as the existing
`swt.Monitor` stub, extended to `Manual`'s type machinery: `MANUAL_TYPES`/`goTypeName`).
`NSWindow`/`NSPasteboard` are additionally `new`'d (`new NSWindow(long)`), so `emitNew` gained a
fallback for `ci == null && Manual.isManual(qualified)` calling a hand-written `New<Type>(id
int64)`. `Protocol` (trivial, `extends id`, no members) and `NSProcessInfo` (2 methods, one of
them `_stret`) needed no stub - translated for real, in the same `j2go` run.

### Other real bugs this surfaced (not direction changes, just bugs)

- **`new T[n]` (sized, no initializer)**: `emitArrayCreation` only ever handled an explicit
  initializer or emitted the bare zero-value literal `T{}` - for `new char[n]` that's a
  **length-0** slice, not one of length `n`. Now `make(goType, sizeExpr)` when a dimension
  expression exists and there's no initializer (`ArrayCreation.dimensions()`). Would have
  silently produced truncated/empty buffers (`NSString.getString()`'s `new char[length()]`)
  with no compile-time signal at all.
- **Reference-type casts** (`(id)other` in `id.equals(Object)`): `emitCast` always emitted Go
  conversion syntax `T(x)`, correct for primitives/structs but not for a translated-class
  target, where Go needs a type assertion (`x.(*T)`) - `T(x)` isn't even valid Go for an
  interface-to-unrelated-pointer "conversion". Now: cast target resolves to a non-struct
  `TypeModel.ClassInfo` -> `expr + ".(*" + goTypeName + ")"`.
- **A parameter/local named after its own enclosing class** (`id.java`'s `id(id id)`
  constructor - completely ordinary in this codebase, `id` names itself throughout): the
  existing constructor template hardcodes `this := &<TypeName>{}`; if a parameter is also
  named `id`, it shadows the package-level type `id` for the rest of the function body ("id
  (parameter) is not a type"). `sanitizeIdent` now also suffixes `_` when a Java identifier
  equals `currentClassGoTypeName` (tracked the same way `currentReturnType` is, saved/restored
  around `emitClass`'s own nested-class recursion).
- **Overload Go-name collisions from JNIGen's generic param names**: `Names.goMemberName`'s
  "later overload gets base+CapitalizedParamNames" rule assumes distinct overloads have distinct
  param *names* - true for hand-written SWT code, false for JNIGen natives, which routinely name
  every overload's extra params `arg0`, `arg1`, ... regardless of type (`objc_msgSend`, ~80
  overloads; `C.memmove`, 15). `nameBasedSuffixesUnique` now checks per overload-group (excluding
  the always-unsuffixed first-declared member, which can't collide since its suffix is never
  emitted) and falls back to `baseName + "Overload" + idx` (declaration position, always unique,
  needs no type information) when it isn't. Regression-tested against stage-1's `Point.OfFloat`
  constructors, which *do* collide by name-only computation (`(int,int)` and `(float,float)`
  both compute "XY") but never actually collide in the emitted output (the first keeps the bare
  base name) - excluding index 0 from the uniqueness check was the fix for that false positive.
- **`java.lang.String`**: 3 more idioms needed beyond what stage-1 exercised - `a.length()` ->
  `int32(len(a))`, `a.equals(b)` -> `(a == b)` (Java's value equality *is* Go's `==` for
  strings), `a.getChars(0, n, dst, 0)` -> `copy(dst, utf16.Encode([]rune(a)))` (every call site
  in this checkpoint copies the whole string into a same-length buffer, so the general 4-arg
  form isn't implemented - only this exact shape). `new String(char[])` ->
  `string(utf16.Decode(buf))`. Java's `array.length` pseudo-field (not a method - different AST
  shape, `QualifiedName`/`FieldAccess` with no real binding) -> `len(array)`.
- **`System.getProperty("os.arch")`** (`OS.IS_X86_64`): only this one property key is
  implemented, calling a hand-written `JavaOsArch()` (`internal/cocoa/platform_manual.go`) that
  maps Go's `runtime.GOARCH` spelling to the JVM's (`"amd64"->"x86_64"`, `"arm64"->"aarch64"`) -
  other property keys still hit the unsupported-marker path.
- **`static {}` block ordering vs. deferred static fields**: `OS.java`'s only static block
  (computing `VERSION` from `NSProcessInfo.processInfo().operatingSystemVersion()`) used to
  become its *own* `func init()`, emitted inline at its position in the class body - textually
  *before* the one combined `func init()` all deferred static-field assignments (`class_x`,
  `sel_x`, ...) are collected into at the very end of the file. Go runs same-file `func init()`s
  in textual order, so the block ran first and read `OSClass_NSProcessInfo`/
  `OSSel_processInfo` before they were set - `objc_msgSend` to a nil class, nil result,
  nil-pointer panic one line later. Now `emitStaticInitBlock` folds its body into
  `deferredStaticInits` too, instead of its own `func init()`; for this file (the block is last
  in source, after every field it reads) that's suffficient - not a fully general fix for a
  static block sandwiched *between* groups of fields elsewhere, which would need interleaving
  the fields-loop and blocks-loop by source position instead of running them as two separate
  passes.
- **One missing native symbol must not take the whole package down**: unlike an
  ordinary/never-called native (fails on its own first call, isolated), a *static field's own*
  deferred initializer runs unconditionally inside the combined `func init()` the moment the
  `cocoa` package is imported - `go test ./...` on this package **is** that first call, for
  every one of ~1500 fields at once. Each deferred entry is now wrapped in its own `func() {
  defer recover(); ... }()`, logged to stderr and left at its zero value on panic, so one bad
  symbol degrades only that one field - this is what caught the `kUTType*` issue below in the
  first place (a clean recovered panic, not a silent hang or a crash that looks unrelated).
- **A zero-arg native whose Dlsym'd "function" is actually a data symbol crashes the process,
  past any Go `recover()`.** The ~100 `kUTType*`/`NSAccessibility*`/`NSPasteboardType*`-shaped
  natives (`kUTTypeFileURL()` etc., no Javadoc flags at all in this file) are JNIGen constant-
  *global* accessors in real SWT output (a hand-written C function returning the address of an
  `NSString * const` framework global) - not ordinary callable functions. With only `libobjc`/
  `Foundation`/`AppKit`/`CoreGraphics`/`CoreText` loaded these simply weren't found (a clean,
  recovered `RegisterLibFunc` panic); adding `CoreFoundation`/`QuartzCore` (direction 1's full
  framework list) made at least `kUTTypeFileURL` resolve to a *real* data symbol somewhere in
  that closure, and calling that address as code is a `SIGBUS` - a hard process crash Go's
  `recover()` cannot catch (`signal arrived during cgo execution`). Fixed *before* the call is
  ever attempted: `emitStaticNativeMethod` now checks `isShadowedByField` (a zero-arg native
  whose name exactly matches a sibling static field - precisely `NSString kUTTypeFileURL = new
  NSString(kUTTypeFileURL())`'s shape, i.e. exactly the `StaticFieldMethodNameClash` cases that
  are also natives) and emits an unsupported-marker stub instead of the generic lazy binding.
  165 of the 166 field/method clashes are natives and now take this path (166th, `VERSION`, is
  an ordinary Java-bodied method, unaffected). This is a heuristic scoped to what's provable
  from this file, not real `flags=const` detection - see "Known gaps".

### Known gaps (not attempted, documented rather than guessed)

- **Only 1 `System.getProperty` key implemented** (`"os.arch"`); any other key hits the
  unsupported-marker path.
- **`String#getChars` only handles the whole-string-copy shape** every current call site uses
  (`0, n, dst, 0`); a partial-range call would silently copy the wrong thing (the 2 middle args
  are ignored entirely, not validated).
- **`org.eclipse.swt.internal.C`** (memmove/malloc/free/strlen/getenv/setenv/`PTR_sizeof`, which
  `OS extends`) is routed into `internal/cocoa`/`cocoa` in `Main.goPackageDir`/`goPackageName`
  rather than getting its own Go package - only `OS.java` uses it, and real cross-package
  qualification (direction 4) has no other consumer yet to justify building against.
  `Platform`/`Library` (`C extends Platform`, `Library.loadLibrary`) are Java's own JNI-library-
  loading bootstrap, not needed at all under purego - stubbed as no-ops (see manual.txt).

## Checkpoint b: every PI/cocoa class in one j2go run

`tooling/port.sh`'s second invocation now requests every file under `Eclipse SWT PI/cocoa` (210
files, `find ... ! -name Selector.java`, sorted) plus `C.java`, instead of the 18-file checkpoint-a
list. All 210 translate; `CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...` pass.
`internal/cocoa/` is now 211 generated files (42317 lines) + 6 manual files (107 lines, most of
them emptied - see below) + 2 test files (51 lines). The translator itself is 2011 lines. The
whole layer needed exactly two translator fixes, both structural, not per-file special cases.

### Constant-global accessors (`emitConstantAccessor`)

The 165 `isShadowedByField` zero-arg natives (`kUTTypeFileURL()`, `NSDefaultRunLoopMode()`,
...) used to fall through to the unsupported-marker stub (checkpoint a: calling one can SIGBUS,
see "Round 2 bugs" above, so it was never safe to just try the generic lazy-native path). Every
one of these carries JNIGen's `/** @method flags=const */` Javadoc right above it in the source
(confirmed by grepping `OS.java`, matches 1:1 with the `isShadowedByField` heuristic already in
place) - this is a real constant-global accessor, not a callable function: the C symbol is the
*address* of the global itself. `emitConstantAccessor` now emits, instead of the panic stub:

```go
func OSKUTTypeFileURL() int64 {
	addr, _ := purego.Dlsym(purego.RTLD_DEFAULT, "kUTTypeFileURL")
	if addr == 0 {
		return 0
	}
	return int64(*(*int64)(unsafe.Add(unsafe.Pointer(nil), addr)))
}
```

`unsafe.Add(unsafe.Pointer(nil), addr)`, not `unsafe.Pointer(addr)`: `go vet`'s `unsafeptr`
check flags a bare `uintptr` -> `Pointer` conversion (confirmed with a 2-line repro under
`go vet`) unless it goes through `unsafe.Add`, which is exempt and is exactly the Go 1.17+
replacement for this idiom. A missing symbol returns 0 rather than dereferencing a nil address -
SWT's own Java code already treats 0 as "not present" everywhere these constants are read.
Dispatches on the declared Go return type: `int64` (every occurrence in this file - all 165
are object/pointer constants read through `long`) reads `*int64`; `float64`/`float32` read
`*float64` (framework globals of this shape are C `double`s regardless of the accessor's own
Java return type) with a `float32(...)` narrowing added for the latter - untested, no `float`-
or `double`-shadowed native exists in the current file set, but the shape is there for one that
would. `internal/cocoa/constant_accessor_test.go` checks `NSDefaultRunLoopMode` resolves to a
non-zero `NSString` whose `UTF8String()` (via `CStrlen`/`CMemmove`) reads `"kCFRunLoopDefaultMode"`.

The per-entry `recover()` around `func init()`'s deferred static inits (checkpoint a) now prints
the field's own Go name (or `<Type> static{}` for a static block) alongside the recovered panic,
not just the panic value - `deferredStaticInitLabels`, parallel to `deferredStaticInits`. With
the constant-accessor fix, `go test ./internal/cocoa/...` now recovers nothing at all (0 stderr
lines), down from panicking on package import before checkpoint a's own fix and printing 165
unnamed recovered panics before this one.

### Wide-hierarchy override false positives (`TypeModel.build`)

The 12-class checkpoint-a set and the id/NSObject/NSString chain are narrow, single-lineage
hierarchies where the override contract ("root struct gets the impl field, `<Root>Impl` lists
methods overridden anywhere, typed with the root's own declared signature") holds by
construction - the root always *is* the class that first declares any name later overridden
below it. Every one of the 210 PI/cocoa classes shares one root (`id`), and unrelated branches
routinely reuse a JNIGen method name by coincidence (dozens of classes each declare their own
`setTitle`, `contentView`, ...) - `NSBox` was the first class hit (2nd js2go run, file 26/210):
`TypeModel.build` recorded *any* two-real-ancestor-pair override anywhere in the whole tree into
`id`'s single shared `overriddenRootMethods` map, keyed only by method name; a class states
`isOverridden(name)` true the moment *any* unrelated pair overrides that name anywhere, then
`findMethodBinding(id.binding, name)` returns `null` because `id` itself never declares most of
these names - `NullPointerException` in `Emitter.paramList`. Fix: skip the registration when
`rootDecl` is `null` (one `if` guard, `TypeModel.java`). A method name family only becomes a
virtual-dispatch candidate when the *shared root itself* declares it (4-5 names: `hashCode`,
`equals`, `toString`, `objc_getClass` - the id/NSObject/NSString chain this was built for);
every other same-name reuse degrades to ordinary Go method shadowing, which is already correct
here since nothing in the PI/cocoa layer holds an ancestor-typed variable and calls a method
expecting a subclass's override (these are leaf value-holder wrappers, always used through their
own concrete type).

### Opaque stubs retired

The 11 opaque `id`-embedding stubs (`NSImage`, `NSWindow`, `NSPasteboard`, `NSOutlineView`,
`NSTableColumn`, `NSCell`, `NSTableView`, `NSArray`, `NSURLAuthenticationChallenge`,
`NSURLCredential`, `DOMEvent`) are gone from `manual.txt`/`Manual.java`'s `MANUAL_TYPES` - all 11
are real generated classes now, referenced by `NSObject.java` (and everything else) exactly like
any other translated type; `stubs_manual.go` is kept as an (empty) file rather than deleted, same
as `sizeof_manual.go`.

### JNIGen Javadoc annotations surveyed, none needed parsing

`accessor=`, `flags=struct`, `flags=cast`, `callback_types=`/`callback_flags=` appear in this
file set (`NSRect.java`, `objc_super.java`, `OS.java`); `flags=const` is the ~166-occurrence
annotation confirmed above. None needed a parser: `flags=struct` is already inferred
structurally (`TypeModel.isStruct`, no annotation needed); `accessor=`/`cast=` on a field
(`NSRect`'s `x`/`y`/`width`/`height` -> `origin.x`/... ; `objc_super`'s `super_class` ->
`swt_super_class`) only tell *JNI* C glue which differently-named/shaped C struct member backs a
Java field - irrelevant here, since a translated struct's Go fields are copied straight from the
Java declaration (already bit-identical to the real ObjC struct - verified for `objc_super` by
checkpoint a's own purego smoke test) and passed by value, so layout, not names, is what purego
needs; `callback_types=`/`callback_flags=` describe the type-descriptor array Java's reflection-
based `Callback` needs to build a JNI trampoline - `cocoa.NewCallback` is a plain, statically
typed Go closure (see "Callback design" above), it needs no separate type descriptor at all.

## Round 3: switch/throw/try-catch, manual superclass embedding, interfaces, SWT.java + events/*

Finished Part 2's remaining statement/expression coverage and used it to translate the target
file list, going past the file set the brief named: `org/eclipse/swt/SWT.java` (previously a
17-line manual stub, now translated for real - 5002 lines, mostly `public static final int`
constants plus 9 methods), `SWTException.java`, `SWTError.java`, `widgets/Listener.java`,
`widgets/Event.java`, `widgets/EventTable.java`, `widgets/TypedListener.java`, and the entire
`org.eclipse.swt.events` package (56 files). `internal/cocoa` is unaffected except for one
translator bugfix rippling through (see "String literal escaping" below) - it was not
re-scoped, still the same 210 PI/cocoa files + `C.java` in one invocation.

### `swt` and `internal/cocoa` invocations restructured (`port.sh`)

Stage-1 (`Point`/`Rectangle`/`RGB`/`RGBA`) used to be its own `j2go` invocation, separate from
stage-2/3's `internal/cocoa` one. `TypeModel` is per-invocation (see "Round 2"), and once
`SWT.java` needed to be a *real* translated class (not a manual stub) instead of `Manual`-only,
every file that references it had to share one invocation with it - so `port.sh`'s first
invocation now also carries `SWT.java`/`SWTException.java`/`SWTError.java`/
`widgets/{Listener,Event,EventTable,TypedListener}.java`/all of `events/*.java`, the same way
checkpoint b merged all 210 PI/cocoa files into one `internal/cocoa` invocation. Confirmed by
grep that nothing already-generated referenced the old manual `SWT` stub's bare-name constants
(`Error`/`ERROR_NULL_ARGUMENT`) before this change, so there was no naming-compatibility
constraint carried over from the stub.

### switch statement (`emitSwitchStatement`, old case:/fallthrough and arrow-rule)

Walks `SwitchStatement.statements()` (a flat list of `SwitchCase`/other-statement nodes)
grouping consecutive `SwitchCase` nodes into one label set (Java's `case A: case B: { ... }`
idiom - multiple empty case labels sharing one body) and the statements after them into that
group's body, until the next `SwitchCase` or the list's end. A label group's Go
`case`/`default` line lists every label's `expressions()` comma-joined (works for the old style
and for Java 14+ `case A, B:` alike). After emitting a group's body: add an explicit
`fallthrough` unless (a) the group is an arrow-rule (`isSwitchLabeledRule()` - Java disallows
fallthrough there by construction), (b) it's the *last* group (Go rejects a trailing
`fallthrough`, and Java just falls out of the switch the same way), or (c) `bodyExits` is true -
the group's last statement (recursing into a trailing `Block`'s last statement) is a
`Break`/`Return`/`Throw`/`ContinueStatement`. A plain (non-labeled) `break`/`continue`/`return`
inside a case body is left as ordinary Go `break`/`continue`/`return` - Go's own semantics for
those inside a `switch` already match Java's. Validated against `TypedListener.handleEvent`
(one 40-arm switch on `Event.type`, some falling through empty label groups, `MouseMove`/
`MouseWheel` arms using `return` instead of `break`) and `SWT.findErrorText`/`SWT.error` (the
grouped-fallthrough idiom with a `//FALL THROUGH` comment between groups - the comment is
irrelevant, JDT sees the same flat `SwitchCase` sequence either way; a final `case
ERROR_NO_HANDLES:` with no trailing `break` correctly falls out of the switch to the method's
own tail code, matching Java, because it's the switch's last group).

`switch` *expression* with `yield` (`emitSwitchExpressionAssign`) is unchanged - still only the
one shape `RGB(float,float,float)` uses, per Part 1/2.

### throw / try / catch / finally / try-with-resources (`emitThrow`, `emitTry`, `emitTryCatch`)

`throw <expr>;` -> `panic(<expr>)` (`emitThrow`), unconditionally - `panic` takes `any`, so this
never needs the target expression to implement Go's `error` interface (values from
`jrt.NewIllegalArgumentException(...)`, for instance, don't - see "Manual superclass embedding"
below).

The finally/resource/catch combination lowers by how many of Go's mechanisms it actually needs,
not one universal shape:
- **`finally` alone, or a try-with-resources with no `catch`** (`EventTable.sendEvent`'s outer
  try: one resource + a `finally`, no `catch`): nothing needs `panic`/`recover`, so the body is
  emitted *inline*, wrapped in ordinary Go `defer`s - no closure boundary at all, so a `return`
  nested arbitrarily deep inside the body (as `sendEvent`'s own `if (event.type == SWT.None)
  return;`, itself inside a `for` inside the try) keeps working exactly like Java's own control
  flow, no special handling needed. `finally`'s `defer func() { <body> }()` is emitted *before*
  each resource's `defer <resource>.Close()`, so Go's LIFO defer order runs resources' `Close()`
  first, `finally` second - the same order JLS 14.20.3 requires (resources close as part of
  completing the try block, before `finally` is considered).
- **A `catch` present** (`EventTable.sendEvent`'s inner `try { listener.handleEvent(event); }
  catch (Error | RuntimeException ex) { exceptions.stash(ex); }`): needs `recover()`, which
  needs a Go closure boundary (an immediately-invoked `func() { defer func() { ... }(); <body>
  }()`) - and that's exactly the boundary a `return`/`break`/`continue` inside the body can no
  longer cross with Go's own keyword. Handled by `emitTryCatch`'s escape scheme: `EscapeScan` (an
  `ASTVisitor`) pre-scans the try body and every catch body for a `ReturnStatement` (always
  escapes, at any nesting depth - it always targets the enclosing Java *method*, not the try) and
  an *unlabeled* `Break`/`ContinueStatement` at `depth == 0` (not inside a `For`/`EnhancedFor`/
  `While`/`Do`/`SwitchStatement` node that itself starts within the scanned subtree - one that
  does stays inside the closure and needs no help). It does **not** descend into a nested
  `AnonymousClassDeclaration`/`LambdaExpression` (a `return` there is its own, not the enclosing
  method's). For whichever of the three the scan finds, `emitTryCatch` declares a `bool` flag
  (plus a named result-shaped local for a non-void return's value) before the closure, has the
  closure set the flag and `return` (from the closure, not the method) instead of doing the real
  thing, and after the closure call, checks the flag and does the real
  `return`/`break`/`continue` - via `returnOrEscape`/`breakOrEscape`/`continueOrEscape`, which
  `ReturnStatement`/`BreakStatement`/`ContinueStatement`'s own emission (in `emitStatementInner`)
  also calls, so nested try/catch compose correctly (an escape out of an inner try/catch, once
  back in the outer try/catch's own context, is itself subject to the outer context's escape
  rules, not hardcoded as a bare `break`). `Emitter.loopSwitchDepth` (a plain counter, incremented/
  decremented by `emitFor`/`emitEnhancedFor`/`emitWhile`/`emitDo`/`emitSwitchStatement`) tracks
  loop/switch nesting *since the innermost active closure boundary*; entering a try/catch's own
  closure resets it to 0 (saved/restored), matching depth tracking to Go closure boundaries, not
  Java lexical scope. Multi-catch (`catch (Error | RuntimeException ex)`) and ordinary catch both
  go through `emitCatchDispatch`: each catch clause's alternatives resolve via `catchGoType` to
  either the broad Go `error` interface (any alternative naming
  `RuntimeException`/`Error`/`Exception`/`Throwable` - see "Manual superclass embedding") or a
  concrete `*T`, and the recovered value is tested with `r.(error)` (comma-ok, for the broad
  case) or an inline `switch r.(type) { case T1, T2: ... }` probe (concrete case, `ex := r` keeps
  `r`'s static `any` type rather than Java's narrower one - not exercised by any file in this
  round, documented as a known gap below); no branch matches -> `panic(r)`, re-raising.
- **Known gap, not implemented**: a labeled `break`/`continue` escaping a try/catch's closure
  (the scan only ever treats *unlabeled* ones as escaping - a labeled one always emits as a
  literal Go `break label`/`continue label`, which fails to compile with a clear error if it
  ever actually needed to escape, rather than silently doing the wrong thing); a `return` inside
  a `finally` block itself (Java's finally-return silently swallows any pending exception/return
  from the try - not attempted, no file in this round's scope has one).

### Manual superclass embedding (`TypeModel.ClassInfo.manualSuperQualifiedName`, `internal/jrt`)

`SWTException`/`SWTError extends RuntimeException`/`Error`, and `TypedEvent extends
java.util.EventObject` - three real, resolvable JDT bindings (JDK bootclasspath), but external,
with no Java source to translate and no `ClassInfo` in `TypeModel`. `TypeModel.build` now
additionally checks, when a class's superclass binding doesn't match any translated `ClassInfo`,
whether `Manual.isManualSuper(qualifiedSuperclassName)` - if so, `ClassInfo.
manualSuperQualifiedName` is set instead of `superclass` staying merely `null`. Three call
sites act on it: the struct-embed line (`emitClass`), the explicit `super(...)` call
(`emitSuperInvocation` - calls the manual type's Go constructor, e.g. `jrt.NewRuntimeException
(message)`, and assigns it to the embedded field), and `super.method()` (a **new** AST case,
`SuperMethodInvocation`/`emitSuperMethodInvocation` - not handled at all before this round;
routes through the embedded field's own Go name, `this.<ManualSuperFieldName>.<GoName>(...)`,
never through `this.<GoName>(...)`, which would just recurse into the override itself). Class
hierarchies with a manual superclass in this round are single-class, leaf-only (no translated
subclass extends `SWTException`/`SWTError`/`TypedEvent`'s *manual* superclass further), so none
of this needed to interact with the impl-cascade/`.impl` machinery at all.

`internal/jrt` (new Go package, hand-written, 60 lines, mirrors `Monitor`/`RoundingMode`'s
manual-type mechanism one level up - a *superclass*, not just a field/param type) provides
`RuntimeException`, `JavaError`, `IllegalArgumentException` (each: `Message string`, a
`New<Type>(message string) <Type>` constructor, an `Error() string`/`GetMessage() string`
method pair, `PrintStackTrace()` that writes the message to stderr; `RuntimeException`/
`JavaError` additionally carry `Cause error` + `Unwrap() error` for `errors.Is`/`As`) and
`EventObject` (`Source any`, `NewEventObject(source any) EventObject`, `GetSource() any`).
Deliberately **not** one shared embedded `Throwable` base for `RuntimeException`/`JavaError`
(each carries its own `Message`/`Cause`) - only two leaves exist, catch-clause dispatch already
type-switches on concrete panic values, and Java's own 4-level `Throwable`/`Exception`/
`RuntimeException`/`Error` hierarchy collapses to a **known ceiling**: a catch of
`java.lang.RuntimeException`/`Error`/`Exception`/`Throwable` by name is translated as a match
against Go's broad `error` interface (`catchGoType`), not against these two types specifically -
so a hypothetical third jrt leaf wouldn't automatically get caught by an existing `catch
(RuntimeException e)` the way Java's real subtyping would. `IllegalArgumentException` was named
`java.lang.IllegalArgumentException` in `Manual`'s registry too (used by `SWT.error`'s `throw
new IllegalArgumentException(message)`) but is **not** in `isManualSuper` - nothing extends it
in the current file set, only `new`s it directly.

**Real bug found and fixed while wiring this up**: naming the Go type for `java.lang.Error`
literally `Error` (mirroring the Java class name, as `RuntimeException`/
`IllegalArgumentException` do) makes `SWTError`'s embedded field - whose implicit name is its
own type's bare name, `Error` - **shadow its own promoted `Error() string` method** (Go's
selector rule: a depth-0 field beats a depth-1 promoted method of the same name), so
`*SWTError` silently stopped satisfying Go's `error` interface (caught by `go build`:
`*SWTError.Error is a field, not a method`, not a runtime surprise). Renamed to `jrt.JavaError`
- the one jrt type whose bare name isn't also its own interface-implementing method's name.

`java.lang.Throwable` (used only as a field/param/return *type* - `SWTException.throwable`,
`getCause()` - never as a manual superclass) maps to Go's builtin `error` interface directly
(`Manual` registers it `isValueType`, spelled bare `"error"`, no `internal/jrt` involvement) -
`.toString()` on a `Throwable`-typed expression is special-cased in `emitMethodInvocation` to
`.Error()` (Go's own string-representation method), ahead of the generic `Manual.isManual`
dispatch (which would otherwise emit a nonexistent `.ToString()`). `java.util.EventListener`/
`org.eclipse.swt.internal.SWTEventListener` (two distinct, content-free marker interfaces -
`TypedListener.eventListener`'s field type, and `TypedListener`'s deprecated `SWTEventListener`-
typed constructor param) both map to bare Go `any`.

### Interface declarations (`emitInterface`)

New: `TypeDeclaration.isInterface()` now branches to `emitInterface` instead of `emitClass`.
Emits `type Name interface { Method(params) ret; ... }` - non-static abstract methods only,
named via the same `names.goMemberName`/`Names.javaMethodBaseGoName` path an ordinary method
uses (see "toString naming bug" below). `superInterfaceTypes()` are embedded by their own Go
interface name when they resolve to a translated `ClassInfo`, silently dropped when they don't
(`java.util.EventListener` extending nothing translatable) - a content-free marker interface
needs no Go embed at all, since it contributes no methods. Static interface methods (Java 8+,
e.g. `ShellListener.shellActivatedAdapter(Consumer<ShellEvent> c)`) go through the same
`emitStaticMethod` package-function path an ordinary static class method uses. Interface-typed
values participate in the existing `instanceof`/cast machinery too, but through a **new,
simpler branch** added to both: `instanceofCheck`/`emitCast` now check `targetCi.isInterface`
first and, if so, emit a direct Go type assertion (`x.(TargetInterface)`, no `*`, no
`.impl`/cascade-helper indirection at all) - the impl-cascade machinery (`.impl` field,
`<root>ImplAs<T>` type-switch helper) exists only for *our own* struct hierarchies and does not
apply to genuine Go interfaces, whose dynamic type is already directly assertable. `ClassInfo`
gained `isInterface` (set in `TypeModel.collect`); `isStruct` is now additionally guarded
`!isInterface` (an interface with zero declared methods would otherwise satisfy the old
`isStruct` heuristic).

Default (body-present) interface methods are not attempted - none exist in the round-3 file set.

### Abstract methods (no body) now skipped instead of crashing (`emitClass`)

A `MethodDeclaration` with `getBody() == null` (an abstract method on an abstract *class*, e.g.
`Layout.computeSize`/`Layout.layout`) used to reach `emitBlockBody(md.getBody(), ...)` with a
null body - not hit by any of the 210+4 previously-translated files (none are abstract classes
with abstract methods), first hit by `Layout.java`. Now skipped outright: nothing is emitted for
the declaring class itself (a concrete override elsewhere provides the real body; if nothing in
the translated set overrides it, the method is simply absent from that Go type, matching how
Java itself never lets you call an abstract method without a concrete override). `Layout.java`
itself is not translated this round (deferred, see "Status" above) - this fix is validated by
compiling cleanly, not yet by a real abstract-class file.

### Anonymous classes: typed correctly, bodies not translated (`emitNew`)

Full anonymous-class method-body translation is not implemented, but a `ClassInstanceCreation`
with `getAnonymousClassDeclaration() != null` (`return new ShellAdapter() { @Override public
void shellActivated(ShellEvent e) { c.accept(e); } };`, `events/*`'s ~25 `xAdapter(Consumer<...>
c)` static factories) is no longer just a bare, generic "unresolved type" marker: `emitNew`
resolves the anonymous class's real superclass (or, absent one, its first declared interface)
to a `ClassInfo` and, if found, emits a panic closure typed to *that concrete pointer type*
(`func() *ShellAdapter { panic(...) }()`) instead of falling back to `any`. This matters because
Go requires a `return`'s expression to be *statically* assignable to the function's declared
return type - a bare `any` is not assignable to a named interface type like `ShellListener`
(only a concrete type that structurally implements it is, and Go checks that at compile time),
so the old generic fallback would have been a compile error here, not a graceful degrade.
`panicClosure` was split into `panicClosure(Expression, message)` (unchanged behavior) and a new
`panicClosureTyped(String goType, message)` it delegates to, letting `emitNew` supply an
explicit type instead of deriving one from `resolveTypeBinding()`. `java.util.function.
Consumer<T>` (the captured lambda parameter's own type, e.g. `c` above) maps to a bare Go
`func(T)` (`GoTypes.map`, new case, reads the one type argument or falls back to `any`) so the
factory method's own *signature* still compiles even though its body doesn't; `.accept(x)` on a
Consumer-typed expression is special-cased in `emitMethodInvocation` to a direct call, `x(arg)`
- not reached by anything in this round (only the *unused*, panic-stub factory bodies reference
`c`), added for when anonymous-class bodies are attempted for real.

### Other real bugs found and fixed this round

- **String literal newline/tab not escaped** (`goStringLiteral`/`escapeForFormat`): a Java
  string literal decoded from a source `"\n     %s"` (`TypedEvent`-derived `toString()` methods
  concatenating a per-line list, e.g. `TouchEvent.toString()`) carries a real newline character
  in `StringLiteral.getLiteralValue()` - emitted verbatim into a Go double-quoted string
  literal, which cannot contain a raw newline (`gofmt`: "string literal not terminated"). Now
  escaped alongside the existing `\`/`"` handling, via a shared `escapeGoQuoted` both functions
  call. Would have hit any string literal in the 210+4 already-translated files with an embedded
  newline/tab too - none happened to have one.
- **`1 << 31`-shaped constants overflow as a Go constant expression** (`foldShift`,
  `emitInfix`): Go's constant-folding checks a shift's *own result* for overflow at compile time
  even when the shift is only ever assigned to a type the folded value fits in (`SWT.java`'s
  `SCROLLBAR_OVERLAY = 1 << 31`-shaped fields - int32's sign bit). A literal-`<<`-literal
  `InfixExpression` whose Go result type is `int32`/`int64` is now folded to its decimal Java
  result directly (`Integer`/`long` wraparound, `shiftAmount & (width-1)`), bypassing Go's own
  `a << b` constant-expression overflow check entirely - correct for every shift amount, not
  just the overflowing ones, so this also fires (harmlessly, same value either way) for benign
  shifts like `1 << 2`.
- **`toString()` named inconsistently depending on override-cascade participation**
  (`Names.javaMethodBaseGoName`, new, replaces 3 duplicated inline switches in `Emitter`): a
  *non-overridden* `toString()`/`equals()`/`hashCode()` got the idiomatic Go spelling
  (`String`/`Equals`/`HashCode`) via a `switch` in `emitInstanceMethod`/`emitMethodInvocation`;
  an *overridden* one (participating in the impl-cascade) got its Go name from
  `TypeModel.build`'s own `Names.capitalize(decl.getName())` - plain capitalize, no special
  case, so cascaded `toString()` was silently named `ToString`, not `String`. Never hit before
  (the 210 PI/cocoa files' only cascaded `toString`, `id#toString()`, is itself
  `manual.txt`-skipped and hand-written in `id_manual.go`) - first hit by `TypedEvent.toString()`
  (`events/*`, the first file set where `toString()` is both cascaded *and* auto-translated), as
  `this.TypedEvent.String undefined` in every subclass's `super.toString()` call. Fixed at the
  root: one `Names.javaMethodBaseGoName(String)` used by both `TypeModel.build`'s cascade-name
  computation and all three `Emitter` call sites (`emitInstanceMethod`, `emitMethodInvocation`,
  `emitSuperMethodInvocation`), so a cascaded and a non-cascaded method by the same Java name are
  now guaranteed to land on the same Go name.
- **A Java local literally named `string`** (`String string = ...;` - a common self-referential
  idiom in exactly these `toString()` methods, 8 occurrences in `events/*` alone) legally
  shadows Go's builtin `string` type for the rest of that function - breaking a *later*
  `func() string {...}` panic closure (an unrelated unsupported call) in the same body
  ("string (local variable) is not a type"). `sanitizeIdent` gained a `GO_BUILTIN_TYPE_NAMES`
  set (`string`, `error`, `any`, `byte`, `rune`, `bool` - the ones a Java identifier could
  plausibly spell) suffixed `_` the same way a Go keyword already is.
- **An unsupported `MethodInvocation`'s receiver/args, once evaluated, were silently discarded**
  (`emitMethodInvocation`'s `ci == null` branch): `args` was always computed (for prelude/side
  effects) before the branch's early return, and the receiver was never even touched - both
  simply vanished from the emitted text. Harmless for a pure expression, but Go (unlike Java)
  makes an unused local variable a hard compile error, and a local referenced *only* inside a
  now-discarded receiver/argument (`String string = ...; return string.substring(...)`, itself
  wrapped in a marker since `substring` isn't implemented) tripped exactly that. Now both are
  blank-assigned (`_ = <recv>`, `_ = <eachArg>`) into `prelude` before the marker, keeping any
  local "used" without affecting the marker's own panic behavior - skipping the receiver when
  the call is `static` (its "receiver" is a type qualifier like `Integer` in
  `Integer.toHexString(x)`, not a value; blank-assigning a bare type name doesn't compile).
- **`array.length` combined with another `int32` was a Go type mismatch** (`emitFieldAccess`/
  `emitQualifiedName`'s `isArrayLength` branches): `len(x)` is Go's untyped-`int`-returning
  builtin, but Java's `array.length` is `int` (`int32` everywhere else in this codebase) - `i <
  types.length` (`EventTable.hook`/`sendEvent`/etc., the first real comparison combining the two)
  is a genuine Go type error, `int32 < int`. Both call sites now wrap `int32(len(x))`.
- **A Java `null` passed/compared where a `String` is expected**: Go's `string` has no nil value
  (`GoTypes.map` already gives it the value type `string`, not `*string`), so the existing
  `"nil"` text for `NullLiteral` doesn't compile as a `string` argument
  (`SWTException(int, null)`-shaped calls) or comparison (`if (detail != null)`, after `detail`
  became a Go `string` param). `adaptNumeric` now rewrites a `"nil"` text to `""` when the
  target type maps to `string`; `emitInfix` does the same for either side of `==`/`!=` against a
  `NullLiteral` when the *other* side's type is `string` (`isGoString`, new).
- **Static method Go name colliding with an unrelated translated class's own name**
  (`staticMethodGoName`, `collidesWithTypeName`, new): `SWT.error(int)`'s computed name
  (`ci.goFuncPrefix + capitalized method name` = `"SWT" + "Error"` = `SWTError`) is *character-
  for-character* the same as the real `SWTError` exception class's Go type name - Go rejects a
  package-level `func`/`type` sharing one identifier. Both `emitStaticMethod` (declaration) and
  `emitMethodInvocation`'s static-call branch (every call site) now go through one shared
  `staticMethodGoName`, which appends `Fn` whenever the computed name matches any translated
  `ClassInfo.goTypeName` (checked against the whole model, not just the declaring class).
  `SWT.error(int)`'s Go name is now `SWTErrorFn`; its 2- and 3-arg overloads (declared later)
  already got parameter-name suffixes from the pre-existing overload rule, so only the
  first-declared overload needed this.
- **`instanceof`/cast against an interface-typed *subject*, not just an interface *target***
  (`instanceofCheck`): the existing rule - "append `.impl` to a subject whose static type is a
  translated class, since that's where its dynamic subtype lives" - fired for `Listener`-typed
  variables too (`EventTable.unhook(int, EventListener)`'s `listeners[i] instanceof
  TypedListener typedListener` - `listeners[i]`'s static type is `Listener`, our own real
  interface, which *is* a `ClassInfo`), producing `this.listeners[i].impl` - but a Go interface
  value has no `.impl` field at all, that indirection exists only for our struct-based
  impl-cascade hierarchies. Now guarded `!subjectCi.isInterface` - an interface-typed subject's
  dynamic value is already directly assertable, exactly like the pre-existing `Object`-typed
  case.
- **`org.eclipse.swt.internal.Platform`/`Library` referenced from *two* Go packages**: both were
  already manual types, but their one hand-written implementation
  (`internal/cocoa/platform_manual.go`) lives in the `cocoa` package, for `C.java`'s sake -
  `SWT.java` (`swt` package) newly calling `Platform.isLoadable()`/`Platform.PLATFORM`/
  `Library.SWT_VERSION` needed its own copies, since Go has no cross-package "same manual
  function" reuse in this design. Added `swt/internal_platform_manual.go`
  (`PlatformIsLoadable() bool`, `PlatformPLATFORM = "cocoa"`) and
  `swt/internal_library_manual.go` (`LibrarySWT_VERSION = 4975`, `4*1000+975` per the real
  `Library.java`'s `SWT_VERSION(major, minor)`/`MAJOR_VERSION`/`MINOR_VERSION` constants).

### System.arraycopy, ExceptionStash, and the widget/graphics stub types

`System.arraycopy(src, srcPos, dst, dstPos, length)` (`emitSystemArraycopy`, new) ->
`copy(dst[dstPos:], src[srcPos:srcPos+length])` - `EventTable.hook`/`remove` both use it.
`org.eclipse.swt.internal.ExceptionStash` (try-with-resources target in `EventTable.sendEvent`,
pulls in `Display.getCurrent()`/`Consumer<RuntimeException>` handlers - `Display` isn't
translated yet) is hand-written, not translated (`swt/internal_exceptionstash_manual.go`,
registered in `Manual`/`manual.txt` like `Monitor`): stores the *first* stashed error and
re-panics it on `Close()`, matching the real class's own eventual behavior once no listener's
exception was handled by a (currently nonexistent) global handler - **simplified**, does not
attempt the real class's "pass to `Display`'s global handler first" step, and does not implement
`addSuppressed` (subsequent stashed exceptions after the first are silently dropped, not
attached) - both documented ceilings, not silent surprises. `Event.java` needed opaque manual
stubs for `Display`/`Widget`/`GC`/`Touch` (`swt/widgets_stubs_manual.go`, empty structs, "Part 3
replaces these") purely as field types (`Event.display`/`.widget`/`.gc`/`.touches`) - none of
`Event`'s own methods call anything on them. `Touch` additionally got a hand-written
`ToString()` (`TouchEvent.toString()` calls `.toString()` on each array element) since a manual
type's method calls are assumed to exist by name (`Manual.instanceMember`), unlike a translated
class's, which are only ever emitted where they're actually declared.

## Round 4: Widget/Control/Scrollable, cross-package qualification

Translated `Widget.java`/`Control.java`/`Scrollable.java` (cocoa, ~8,170 lines combined) for
real. `internal/cocoa` (checkpoint b's own invocation) is untouched in content; `swt`'s
invocation now also parses all 210 PI/cocoa files + `C.java` as **reference-only** input.

### Reference-only files (`Main.java`, `--`)

`j2go`'s CLI now takes `<emit files...> [-- <reference-only files...>]`. Files after `--` are
parsed and fed into `TypeModel.build`/`Names` exactly like emit files (so their `ClassInfo`s,
overload-order state, and exact Go names all resolve identically to when they were *actually*
translated - `Names`' overload disambiguation depends only on a declaring class's own source
order, not on what else is in the invocation, so this is guaranteed byte-identical), but
`Main.main`'s emission loop only iterates the emit-file prefix. This is how `swt/widgets_*.go`
resolves `cocoa.OS.objc_msgSend`, `cocoa.NSView`, etc. by their real, already-generated names
without a second copy of `internal/cocoa` being written. `port.sh`'s first invocation now reads:
stage-1/round-3 files, `Widget.java`/`Control.java`/`Scrollable.java`, `--`, `C.java` + all 210
cocoa files.

### Cross-package qualification (`Emitter.qualify`/`qualifiedTypeName`/`qualifiedFuncPrefix`)

`TypeModel.ClassInfo` gained `javaPackage`/`goPackage` (set in `collect()` via
`GoTypes.goPackageOf`, the same java-package-\>go-package routing `Main.goPackageDir` already
used, now exposed as a public static so both places share it). `Emitter` gained
`currentJavaPackage`/`currentGoPackage` (set per compilation unit) and:

- `qualifiedTypeName(ci)` / `qualifiedFuncPrefix(ci)` / `qualify(bareIdent, ci)`: prefix with
  `"cocoa."` + register the import when `ci.goPackage != currentGoPackage`, else bare. Every
  `GoTypes.map` call site (28 of them) now takes `Emitter` instead of `TypeModel` so `map` can
  route its `ci != null` branch through `qualifiedTypeName`; every other place that built a name
  straight from `ci.goTypeName`/`ci.goFuncPrefix` (static field/method refs, `new` construction,
  casts, instanceof, the cascade-helper generator) now goes through one of these three instead.
- **Guard** (`packagePrefix`): qualifying a type when `currentGoPackage == "cocoa"` prints
  `j2go: guard violated: cocoa file ... must not reference a <pkg> type` and exits 1 - cocoa must
  never import swt. `checkNoForeignPackageLeak` (called from `GoTypes.map`'s final
  `unsupported_type_` fallback) covers the same case for a type that isn't even in the model.
  Neither path is exercised this round (cocoa's own invocation never sees an swt type), but both
  are real, not aspirational - a future cocoa-side regression would hard-fail immediately with a
  clear message instead of silently emitting `unsupported_type_org_eclipse_swt_...`.
- **Lowercase Java class names** (`id`, `objc_super` - Cocoa's own naming, not ours): a bare
  `Names.capitalize` on a cross-package reference produces `Objc_super`, not the hand-written
  alias's `ObjcSuper` (`internal/cocoa/id_manual.go`'s `type Id = id` / `type ObjcSuper =
  objc_super`, pre-existing). `Emitter.LOWERCASE_ALIASES` is a 2-entry exact-name map consulted
  before falling back to `Names.capitalize`, only when actually crossing packages.
- **The embedded-field selector case** (`upcastObject`, see below): a promoted embedded field's
  Go name is fixed to its type's own declared spelling (`x.id`, lowercase, unexported) -
  no alias, however spelled, makes `x.Id` mean "the embedded id value" (it already means the
  *promoted* `Id int64` field one level down). `id_manual.go` gained `func (this *id) AsId() *id
  { return this }`, a promoted *method* (unaffected by the embedding field's own name/export
  status); `upcastObject`'s target-is-`id` case emits `text + ".AsId()"` instead of
  `"&" + text + ".id"`. Nothing else in the translated set needs this - `objc_super` is never an
  upcast target.

### `upcastObject` (`ExpressionEmitter`): one mechanism for every object-typed target mismatch

Generalizes the return-only `upcastForReturn` (Round 3-era, now deleted) to every place Go needs
an explicit conversion Java does implicitly for a *narrower object type assigned/passed/returned/
compared where a wider one is declared*: `adaptNumeric` now calls it for any non-primitive
`(from, to)` pair (covers assignment, `var` init, ternary arms, array elements, method
arguments - every existing `adaptNumeric` call site gained this for free), `emitReturn` (dropped
its separate `upcastForReturn`+temp-var dance - `&expr.Field` is valid Go directly, even when
`expr` is a complex expression, since dereferencing *any* pointer is addressable regardless of
how it was obtained), `emitInlineAssign`, and `emitInfix` for `==`/`!=` between two related
non-array object types (`this == shell`, `*Control` vs `*Shell` - whichever side's `upcastObject`
call actually changes text is the narrower one).

Walks both the real `TypeModel` superclass chain and a **new** manual-hierarchy chain
(`Manual.manualSuperclassOf`/`Manual.WIDGET_SUPER`: `Composite`\-\>`Scrollable` (real), `Canvas`\-\>
`Composite`, `Decorations`\-\>`Canvas`, `Shell`\-\>`Decorations`, `Menu`\-\>`Widget` (real),
`ScrollBar`\-\>`Widget` (real) - the widget-hierarchy manual stubs Round 4 needed embed these same
types as their first field, so the promoted-field upcast (`&x.Control`) actually resolves).
`isProperDescendant`/`superQualifiedOf`/`bareGoName` do the walk by qualified-name string,
capped at 12 hops (the real hierarchy is 6 deep at most).

### Other translator fixes this round

- **`emitInfix`**: `~x` (bitwise complement) now maps to Go's `^x` (was hitting the generic
  unsupported-PrefixExpression marker). Two Java arrays compared with `==`/`!=` (reference
  identity in Java, but a Go slice can only compare to `nil`) now emit `slices.Equal(...)` (or
  its negation) - a full compare is a harmless superset of what the identity check was a fast
  path for.
- **Switch case labels** (`emitSwitchStatement`): a case constant narrower than the switch
  subject (`case SWT.ESC:` - a Java `char` - inside `switch(int keyCode)`) now goes through
  `adaptNumeric` per label, matching how a binary operand already would. Go's typed `const`s
  don't get Java's implicit widening for free.
- **Casts off `Object`** (`emitCast`): any cast whose *source* Java type is `java.lang.Object`
  (Go `any`) now emits a type assertion (`expr.(T)`) instead of a conversion, for every target
  kind - not just the translated-class/manual-class cases already handled, but primitives,
  strings and arrays too (`(Object[]) obj` -\> `obj.([]any)`). Go's `T(x)` conversion syntax
  doesn't apply from an interface to a concrete type at all; only the already-existing
  class/manual branches happened to route around this before.
- **`emitCast`/`instanceofCheck`, the real-`ClassInfo` branch**: fixed the same "concrete pointer
  isn't assertable" bug the manual-type branch already handled - a cast/instanceof whose
  *subject* is one of our own struct types (not an interface) needs `.Impl` first
  (`this.View.(*cocoa.NSControl)` was invalid Go; `this.View.Impl.(*cocoa.NSControl)` is).
  Factored into one shared `implSubject(text, expr)` used by both `emitCast` and
  `instanceofCheck`.
- **`super.method()` and the collision-rename cascade** (`emitSuperMethodInvocation`): a method
  under the impl cascade can have been renamed to `<Base>On<Class>` to avoid a same-bare-name
  collision elsewhere in the tree (`TypeModel.build`'s existing disambiguation, e.g.
  `setZOrder()`/`updateCursorRects(boolean)`, each declared on `Control` with an unrelated
  `(Control, boolean)`/`(boolean, NSView)` overload too). `super.x()` bypasses the cascade's
  `.Impl` dispatch by design (calls the specific ancestor directly) but still has to call it by
  its *actual emitted name* - it didn't before, producing `this.Control.SetZOrder()` when the
  real method was `SetZOrderOnControl`.
- **Implicit constructors** (`ClassEmitter.emitImplicitConstructor`, JLS 8.8.9): a class with no
  declared constructor at all (`Event.java`, `EventTable.java`: field declarations only) used to
  get nothing emitted - `new Event()` elsewhere had no `NewEvent()` to call. Now synthesizes the
  same `New<X>()`/`init<X>()` split `emitConstructor` produces for an explicit one (so a
  subclass's own explicit `super()` can still find `init<X>()`), with the implicit constructor's
  own accessibility matching its class's (JLS 8.8.9) - `EventTable`'s class is package-private,
  so its implicit ctor is `newEventTable`, not `NewEventTable`. Both this and
  `emitConstructorBody`'s own zero-arg-super-call case now share one `emitZeroArgSuperInitCall`
  helper (previously duplicated).
- **C-style array declarations** (`emitVarDecl`): `Touch touches[] = new Touch[n];` (the `[]`
  binds to the *fragment*, not the shared type node - legal, if archaic, Java) used to read the
  declaration's own type node (`Touch`, no array) instead of each fragment's resolved type
  (`Touch[]`), silently declaring `*Touch` instead of `[]*Touch`. Now resolves each fragment's
  own type via `VariableDeclarationFragment.resolveBinding().getType()`.
- **`GoTypes.map`**: cross-package qualification threaded through (see above); otherwise
  unchanged.
- **`JdkIntrinsics`**: added `Math.abs` (float/double via `math.Abs`, int/long via a branch - Go's
  `math` package has no integer `Abs`), `String.substring(int[, int])` (byte-slicing - not
  UTF-16-index-correct for non-ASCII, fine for the ASCII class-name use in `Widget.getName()`),
  `String.lastIndexOf(char)` (`strings.LastIndexByte`), `Integer.toHexString` (unsigned 32-bit,
  matching Java's own semantics, via `strconv.FormatUint`), `Boolean.valueOf` (both overloads -
  the `boolean` one is a no-op box, the `String` one is `strings.EqualFold(s, "true")`),
  `Objects.equals` (only the two-Go-string shape used here, `==`), `System.getProperty` for any
  key besides `"os.arch"` (reads as `""`/unset, matching the real JDK's behavior for a key that
  was never set), `Thread.currentThread()` (`ThreadCurrentThread()`, pairs with `Display.thread`
  being `any` - both sides are always `nil`, so `checkWidget`'s thread-affinity check always
  passes rather than always panicking; see manual.txt's `java.lang.Thread` entry).
- **Generic methods** (`emitClass`, `MethodDeclaration.typeParameters()`): a generic method
  (`Widget.getTypedListeners<L extends EventListener>`, uses `Stream`/method references, neither
  supported) has no Go equivalent for its own *signature*, not just its body - skipped entirely
  at its declaring class, the same treatment an abstract (no-body) method already got, rather
  than emitted with an `unsupported_type_...` return type that would break `go build`.

### Manual stubs added this round

All in `swt/`, registered in `tooling/j2go/manual.txt` and `Manual.java`'s `ENTRIES`/
`WIDGET_SUPER`. `Widget` itself is **removed** from both (translated for real).

| Type | File | Members | Superclass embedded |
|---|---|---|---|
| `Composite` | `widgets_manual_stubs2.go` | `RemoveControl`, `_getChildren`, `_getTabList` (genuinely not declared anywhere in the translated set - everything else `Control.java` calls on a `*Composite` promotes from the embedded `Scrollable`) | `Scrollable` (real) |
| `Canvas` | ″ | none beyond embedding | `Composite` |
| `Decorations` | ″ | `SetSavedFocus`, `FixDecorations`, `BringToTop` | `Canvas` |
| `Shell` | ″ | `BringToTop` (own override), `SendToolTipEvent`, `GetModalShell`, `Layout`, `FixShell`, `SetActiveControl` (variadic trailing `int32` covers both real overloads - `Manual.instanceMember` has no overload awareness), `GetShell` (overrides `Control`'s climb-to-parent version to stop at itself - without it, the promoted `Control.GetShell()`, `this.parent.GetShell()`, recurses until `parent` is nil) | `Decorations` |
| `Menu` | ″ | `SetLocation`, `SetVisible` (`Dispose`/`IsDisposed` promote from `Widget`) | `Widget` (real) |
| `ScrollBar` | ″ | fields `parent *Scrollable`, `view *cocoa.NSScroller`, `target *cocoa.Id`, `actionSelector int64`; `NewScrollBar`, `UpdateBar`, `SendSelection` | `Widget` (real) |
| `GCData` | ″ | every field `Control.java`'s own `internal_new_GC`/`internal_dispose_GC` read/write; `PaintRect` is `any` not `cocoa.NSRect` - assigned a plain value *and* null-checked, and a bare Go struct can't compare to `nil` | - |
| `Font`/`Color`/`Image`/`Cursor`/`Region` | ″ | `Handle` field + `IsDisposed`; `Font` also `ExtraTraits`; `Color` also `GetAlpha`; `Image` also `GetBounds` (panics - not translated); `Font`/`Color`/`Region` also their `cocoa_new` constructors | - |
| `Accessible` | ″ | the dozen `Internal_accessibility*`/`Internal_addRelationAttributes`/`Internal_dispose_Accessible` bridge methods `Control.java`'s own `accessibility*` overrides call, all no-op/false/nil (matches real SWT's own behavior for a `Control` with no screen-reader `Accessible` attached) | - |
| `ACC` | ″ | `ACCCHILDID_SELF = -1` (only member read) | - |
| `WidgetSpy` | ″ | `WidgetSpyIsEnabled` (var, starts false), `WidgetSpyGetInstance()`, `WidgetCreated`/`WidgetDisposed` (no-op) | - |
| `Callback` | ″ | `NewCallback`/`GetAddress` panic, `Dispose` no-op - reflection-based `(Object, methodName, argCount)` dispatch has no Go equivalent (distinct from `cocoa.NewCallback`'s real closure mechanism), unsupported marker by design, not this round's job | - |
| `Dialog` | `widgets_stubs_manual.go` | none (only ever compared to `nil`) | - |
| `Device`/`Resource` | ″ | none - referenced only as an *inherited-method declaring class* (`display.getSystemFont()`, `someFont.dispose()`), never their own declared variable type; aliased to `Display`/`any` rather than adding distinct unused types | - |
| `AutoscalingMode` | ″ | `type AutoscalingMode int32` - a Java enum (still no translator rule for those), the one setter that takes it ignores its argument | - |
| `Display` (expanded) | ″ | ~35 fields/methods: event/focus/touch/gesture bookkeeping, `GetSystemFont`, `Map`/`MapRect`, `IsActivateShellOnForceFocus`, etc. - field/method *shapes* only, no real event loop |  |
| `GC` (expanded) | ″ | `Handle` (was unexported `handle` - `Control.java` reads it as `gc.Handle`, a real public Java field), `IsDisposed`/`Dispose` | - |
| `Touch` (expanded) | ″ | real fields (`Identity`/`Source`/`State`/`Primary`/`X`/`Y`) + `NewTouch` (was a 0-field opaque stub) | - |
| `Monitor` (expanded) | `widgets_monitor_manual.go` | `GetBounds` (zero `Rectangle` - `Control.getMonitor()`'s area/intersection math needed it) | - |

### `internal/cocoa/id_manual.go`

Gained `AsId()` (see "cross-package qualification" above) - the only change to this package this
round; `internal/cocoa`'s own generated output is unchanged (see "reference-only files").

### Component sizes

`ClassEmitter.java`/`ExpressionEmitter.java` both grew past the 450-line budget from this
round's additions; both dropped small pre-existing duplication (`bareGoName`/`superQualifiedOf`
shared one `findByQualifiedName` helper; `emitConstructorBody`'s and the new
`emitImplicitConstructor`'s zero-arg-super-call logic share `emitZeroArgSuperInitCall`;
`emitExprHoisted`, dead code, deleted; a redundant `if`/`else` returning the same expression
twice in `emitQualifiedName` collapsed) plus comment compression to land back at 449/449.

## Round 5: Composite/Canvas/Decorations/Shell/Button, layouts (former status block)

**Round 5: done, verified, green.** `mvn -q -f tooling/j2go/pom.xml package && bash
tooling/port.sh && CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...` all pass. 33 Go
tests pass (30 from before + 3 new, `swt/layout_test.go`). `internal/cocoa` content unchanged
(still all 210 PI/cocoa files + `C.java` in one invocation) except the `Impl` -> `impl` rename
(below) touching every file's constructor. `swt`'s own invocation grew to include
`Layout`/`Item`/`FillLayout`/`FillData`/`RowLayout`/`RowData`/`Composite`/`Canvas`/`Decorations`/
`Shell`/`Button` alongside Round 4's files.

**Housekeeping done first**, per the round's own brief:
- **`impl` field unexported** (was `Impl`, leaking into the public API). Same-package code reads
  `x.impl` directly; cocoa's `id` cascade (the only one crossed from `swt`) gained an exported
  accessor, `func (this *id) Impl() idImpl { return this.impl }` (`internal/cocoa/id_manual.go`,
  next to the existing `AsId()`). The translator now threads this through generically via
  `Emitter.implAccess(rootCi)` (`".impl"` in-package, `".Impl()"` cross-package), used at every
  emission site that used to hardcode `.Impl` (`ClassEmitter`'s field decl + `this.Impl = this`,
  now in `ConstructorEmitter`; `InvocationEmitter`'s override-cascade call + covariant-downcast
  temp; `TypeTestEmitter.implSubject`). Fixing this exposed a real, pre-existing latent bug: the
  covariant-downcast temp in `InvocationEmitter` used the *calling* method's cascade root
  (`ci.root`) for its own `.impl` access, but the temp's actual Go type is the *target* (covariant
  return type)'s root, which can be in a different package - `Widget.topView()` returns
  `cocoa.NSView`. Fixed to use `target.root`.
- **`swt/widgets_manual_stubs2.go` renamed** to `swt/widgets_stubs2_manual.go` (the `*_manual.go`
  suffix rule).
- **`ClassEmitter`/`ExpressionEmitter` split**, each pulling out one real responsibility: the
  constructor/instance-initializer/super-this-dispatch code moved verbatim to a new
  `ConstructorEmitter` (`emitConstructor`, `emitImplicitConstructor`,
  `emitZeroArgSuperInitCall`/`emitConstructorBody`/`emitInstanceInitializers`,
  `emitSuperInvocation`/`emitThisInvocation`/`emitSuperMethodInvocation`); the infix/numeric-
  widening/upcast/zero-value code moved verbatim to a new `NumericEmitter` (`emitInfix` +
  `adaptNumeric`/`upcastObject`/`zeroValue` and their private helpers). Verified as a pure move: a
  snapshot of `swt/`+`internal/cocoa/` taken before the split, regenerated after, `diff -r` clean
  (done in two steps - split first with `Impl` still capitalized, diffed clean; the `impl` rename
  applied and reverified via the full test suite separately, not by another byte-diff since it's
  an intentional content change). `ClassEmitter` 290 lines, `ExpressionEmitter` 246 (pre-lambda
  work) then 293 (after adding `ExpressionMethodReference` support, see below);
  `ConstructorEmitter` 187; `NumericEmitter` 231 then 252 (after the 3+-operand arithmetic-chain
  fix, see below). All emit components stay under the 450-line budget with room to spare.

**Widget-hierarchy classes translated for real**: `Composite`(1324 lines)/`Canvas`(699)/
`Decorations`(718)/`Shell`(2580)/`Button`(1072) (cocoa), `Layout`(107)/`Item`(237) (common
widgets), `FillLayout`(254)/`FillData`(48)/`RowLayout`(522)/`RowData`(129) (layout) - ~7,690 lines
of Java, all removed from `manual.txt`/`Manual.ENTRIES`/`Manual.WIDGET_SUPER` where they used to
be manual stubs (`Composite`/`Canvas`/`Decorations`/`Shell`). `Layout` is its own impl-cascade
root (not part of Widget's - it has no superclass at all); `FillLayout`/`RowLayout` override its
two abstract methods (`computeSize`/`layout`) for real. The dispatch-through-`.impl` mechanism now
demonstrably serves a *second* real hierarchy split across files: `Control.ComputeSize` (
`swt/widgets_control.go:461`) calls `this.impl.ComputeSizeWHintHHintChangedOnControl(...)`, which
resolves at runtime to `Composite`'s (`widgets_composite.go:159`) or `Shell`'s
(`widgets_shell.go:304`) own override, never Control's generic default, exactly the polymorphism
the impl-cascade exists for.

**`ExpressionMethodReference` (`obj::method`) translated for real** - the only lambda/method-ref
shape on this round's path (`Item.java`'s `this::handleDPIChange`, bound to the `Listener`
functional interface; zero raw lambdas anywhere in Composite/Canvas/Decorations/Shell/Button/
Layout/Item/FillLayout/FillData/RowLayout/RowData). Go has no bound-method-value-to-differently-
named-interface-method coercion, so a small adapter type is generated once per target interface
(`ExpressionEmitter.emitMethodReference`/`ensureFuncAdapter`, reusing the existing
`fileHelperSource`/`generatedHelpers` mechanism `TypeTestEmitter`'s cascade helpers already use):

```go
type ListenerFunc func(event *Event)
func (f ListenerFunc) HandleEvent(event *Event) { f(event) }
```

then `this::handleDPIChange` becomes `ListenerFunc(this.HandleDPIChange)`. A plain
`LambdaExpression` (none exist on this round's path) is **not** implemented - it would still hit
the pre-existing generic "unsupported" catch-all, which already degrades gracefully to a typed
panic closure (`GoTypes.map` already resolves a functional-interface target type correctly, per
Round 3's "Anonymous classes" note) rather than a hard compile break. Display's own 46 lambdas
(next round) are unexamined - this round did not look at Display.java's lambda *shapes* at all,
only confirmed the mechanism `ExpressionMethodReference` needs (a generated func-adapter type per
target interface) is real and works; a `LambdaExpression`'s own body (a `Block` or a bare
expression) would need a new case in `emitExpr` building a Go func literal the same adapter wraps
- not attempted, no file in this round needed it.

**7 further translator bugs found and fixed**, each a real, general issue exposed by files outside
the narrow set Rounds 1-4 exercised, not one-off patches (full detail in "Round 5" below):
1. `TypeTestEmitter.ensureCascadeHelper`'s naming assumed `target`'s Go prefix always textually
   starts with `root`'s (true only for the nested-class case, e.g. `Point`/`Point.OfFloat`) -
   crashed (`StringIndexOutOfBoundsException`) the first time an unrelated top-level pair (root
   `Widget`, target `Shell`, `topView()`'s covariant return) hit it.
2. `ConstructorEmitter.emitSuperInvocation`/`emitThisInvocation` built their args directly into a
   `StringBuilder`, before any enclosing `emitStatement` had set up `Emitter.prelude` - a
   ternary/instanceof in a `super(...)`/`this(...)` argument (`Shell`'s delegating constructors)
   crashed on a null `List`.
3. `resolveCrossFamilyNameCollisions` skipped every cascade (override-point) method on the
   assumption a non-cascade sibling can't collide with one - true until `Button.java` newly
   overrides `Control.setBackgroundColor(NSColor)`, making it a cascade participant for the first
   time and exposing that it collides with `Control.setBackground(Color)`'s own overload-suffixed
   name. Fixed by seeding the per-class collision set with cascade names too.
4. A cascade method's Go name can equal a class's own bare name somewhere in its tree
   (`Layout.layout()` -> `"Layout"`, the same name every subclass's *embedded* `Layout` field
   already uses) - Go rejects a struct with both a field and a method of that name.
   `TypeModel`'s cascade-naming pass now suffixes `Fn` when this happens.
5. Chained assignment (`child = update[i] = composite;`) never adapted types between steps,
   assuming (true only in the 3 previously-exercised files) every target in a chain shares one
   type - `Composite.layout`'s traversal chains `Composite` through an intermediate array slot up
   to a `Control`-typed target, needing an upcast Round 1-4 never had reason to apply mid-chain.
6. `(Display) null` (a disambiguating cast Java needs to pick a constructor overload, not a
   runtime check) emitted `nil.(*Display)` - an invalid Go type assertion on a bare `nil`. Now
   short-circuits to bare `nil` whenever the cast's *operand* is itself `NullLiteral`.
7. A manual (untranslated) type's own field/method access always assumed the *declaring class of
   the resolved Java method* determines dispatch - true until a manual leaf type (`Caret`/`IME`)
   *inherits* a method from a real translated ancestor (`Widget.release`/`.sendEvent`): the
   receiver has no `.impl` field, and a bare `Manual.instanceMember` capitalize picks the wrong
   Go name for an overloaded inherited method (`sendEvent(int)` is really
   `SendEventEventType`, not `SendEvent`). `InvocationEmitter.emitMethodInvocation` now checks
   the *receiver's* own static type for this case, computing the Go name exactly as the real
   declaring class would (cascade-aware) and calling it as an ordinary promoted method (no
   `.impl`, since the manual receiver isn't itself a cascade member).

**2 smaller generic wins**, not bug fixes: `String.trim()` -> `strings.TrimSpace` (`JdkIntrinsics`,
used by `RowData.toString()`), and a fix to `NumericEmitter.adaptNumeric`/`emitInfix` so a 3+
operand arithmetic/bitwise chain (`imageRect.X + imageRect.Width + ButtonIMAGE_GAP`) widens every
operand to the chain's own Java-resolved result type, not just the first pair (`adaptBinaryOperands`
only ever compared two operands - the third silently kept its own narrower type, a Go compile
error whenever it differs).

**`Main`'s sourcepath gained `Eclipse SWT/emulated/bidi`** (`Item.java` references `BidiUtil`,
which only exists there and in `win32/` - confirmed via the real cocoa build fragment's own
`build.properties`, which pulls in `emulated/bidi` alongside `common`/`cocoa`). `BidiUtil` itself
is not translated (still hits the generic unresolved-call marker) - only made *resolvable* so
parsing doesn't abort with a binding error.

**Manual stubs extended, not newly invented** - every addition is real fields/methods on an
*existing* manual type (`Display`, `GC`, `Monitor`), or a new *leaf* manual type for something
Composite/Canvas/Decorations/Shell/Button hard-require and nothing else translates yet: `Caret`,
`IME`, `MenuItem`, `ToolBar`, `ImageData` (all named explicitly in the round's own brief), plus
`GCData.Image`. `java.lang.Integer`/`java.util.Map`/`java.util.HashMap` are new *value-type*
entries mapping to bare `any` (Shell's AWT window-embedding bookkeeping, `windowEmbedCounts`, is
the only user - dead code until Display's AWT bridge exists; every read/write on it already
degrades to an unresolved-call panic). Registering them exposed one more real gap: a value type
mapped to bare `any` has no actual Go methods, so `Manual.isManual`'s existing instance-method
dispatch (bare capitalize, assumes a method exists) doesn't apply to it - `Manual.isBareAny` now
excludes it, falling through to the ordinary unresolved-call degrade instead
(`InvocationEmitter.emitMethodInvocation`, two call sites). Likewise `emitNew`'s
`Manual.isManual` branch assumed every manual type has a real Go constructor function
(`Manual.ctorFuncName`) - untrue for a value type (`new HashMap<>()` needs no constructor call at
all, just `any`'s zero value, `nil`) - fixed the same way `zeroValue` already handles it elsewhere.
Two genuinely overloaded manual-declaring-class methods needed per-signature `MANUAL_METHODS`
entries, same pattern as Round 4's `Display.map`/`.mapRect`: `Display.getWidget(long)`/
`.getWidget(NSView)` -> `GetWidgetById`/`GetWidgetByView`, `Display.findControl(boolean)`/
`.findControl(boolean, NSView[])` -> `FindControl`/`FindControlHitView`.

**New: `tooling/j2go/names.properties`** (loaded by `Names.loadOverrides`, previously an unused
hook with no file to load) - one entry, `Button.createString()` (a genuinely new, 0-arg method
Button declares locally) renamed to `CreateAttributedTitle` to stop it shadowing the *promoted*
7-arg `Control.createString(...)` Button also calls: Go has no overloading, so a subclass's own
same-named-different-arity method always shadows an inherited one regardless of arity, breaking
every 7-arg call site. Same root cause as translator bug #3 above (a same-bare-name reuse across
an inheritance boundary), but this one is a single, real, intentional Java overload naming
collision (not a translator gap) - the existing hand-pin mechanism is the correct fix, not a
generalized rule. One more instance, `Canvas.drawBackground(GC,int,int,int,int)` (shadows
`Composite`'s promoted 7-arg overload the same way) -> `DrawBackgroundGC`, pinned the same way.

**Not attempted**: `GridLayout`/`FormLayout` (explicitly optional this round) - skipped outright,
not even tried. Given how many real translator bugs the *much smaller* `FillLayout`/`RowLayout`
pair surfaced (bugs 3-6 above), attempting `GridLayout`(754 lines)/`GridData`(577)/
`FormLayout`(391)/`FormData`(348) - 2,070 more lines, a materially larger and more field-heavy
API - would very likely need more translator work too, which the round's own instruction only
authorized "if they go through without new translator work". `synchronized`, `java.util`
collections generally (beyond the narrow `any`-mapped `Map`/`HashMap` above), `Synchronizer.java`/
`RunnableLock.java` - same as every round so far, still true. Widget's `getTypedListeners<L>` -
still skipped (unchanged from Round 4). `Control`'s `new Callback(...)` - still a manual-stub
panic (unchanged).

**Next round: `Display.java`.** 6861 lines, currently a ~50-member manual stub
(`swt/widgets_stubs_manual.go`, grown again this round - `GetWidgetById`/`GetWidgetByView`,
`CreateWindowSubclass`, `CheckFocus`, `CascadeWindow`, `ClearModal`, `GetPrimaryMonitor`,
`GetShells`, `SetMenuBar`, `SetModalShell`, `UpdateQuitMenu`, `RunLoopModes`,
`FindControl`/`FindControlHitView`, `GetMenus`, `_getFocusControl`, `UpdateDefaultButton`,
`IsBundledIconSet`, `IsDisposed`, `AddLayoutDeferred`, `GetNSColorRGB`, plus fields `keyWindow`/
`appMenuBar`/`clickCountButton`/`escAsAcceleratorPresent`/`modalPanel`/`modalShells`/
`systemUIMode`/`systemUIOptions`/`dockImage`/`application`/`disposed`, and the 2 static functions
`DisplayGetCurrent`/`DisplayGetDefault` both still returning `nil` - **no live Display exists
yet**, exactly why a window can't actually open until this is translated for real). Order: (1)
`Synchronizer`/`RunnableLock` (`Display`'s own event-loop plumbing needs them, "not attempted" in
every round so far, including this one); (2) `Device`/`Font`/`Color` for real (currently thin
manual stubs Round 4/5 only shaped to compile, not to work); (3) `GC`/`Image` only as far as
`Display`'s own init/paint path needs; (4) callbacks - `Display`'s ~46 lambdas are the first real
lambda-body-translation workload (`ExpressionMethodReference` exists now, see above; a
`LambdaExpression`'s own body does not - expect this to be the round's biggest new-construct
lift); (5) `cmd/snippet` - a minimal `Shell`+`Button`+`readAndDispatch` loop, the task's actual
end-to-end goal, only reachable once (1)-(4) exist.

## Round 6: Display, the event loop, callbacks - contract changes

Every item is a general translator rule unless it names a manual file.

- **Functional values** (`FunctionalEmitter`). A lambda/method reference/one-method anonymous
  class becomes a Go func literal wrapped for its target type: `java.lang.Runnable` ->
  `jrt.Runnable` interface, created as `jrt.NewRunnable(func() {...})` (a pointer, so
  `timerExec`'s `==` lookup compares identity); `Consumer<T>` -> bare `func(T)`; a translated
  interface -> the generated `<Iface>Func` adapter, now a **struct pointer** `&ListenerFunc{fn:
  ...}` instead of a named func type (a func inside an interface panics on `==`, and
  `EventTable.unhook` compares listeners). A lambda body gets fresh return type/escape/loop state
  (`Emitter.enterFunctionBody`). An anonymous class's `this` is its own holder variable, declared
  before the body so `timerExec(rate, this)` works; an unqualified member it inherits (and the
  outer class doesn't) resolves to that variable too (`Emitter.implicitThis`).
- **Anonymous subclasses** of a translated class (the 25 `events/*Listener` adapters) become a
  generated `<Class>Anon<N>` struct embedding the base, one `fn<Method>` field per overridden
  method plus a forwarding method; created in the prelude (`impl` set, base `init` called).
  Ceiling: the type isn't in the impl cascade, so it is only reached through an interface.
  `Runtime.addShutdownHook(new Thread(){...})` is dropped (no Go equivalent).
- **`synchronized`** -> one process-wide reentrant monitor (`internal/jrt/lang.go`,
  reentrancy by goroutine id), entered before and released by a `defer` inside a block-scoped
  closure that reuses the try/catch escape machinery (`ControlFlowEmitter.emitClosure`).
  `wait`/`notify`/`notifyAll` map onto it. Ceiling: unrelated blocks serialize.
- **`java.util` containers**: `Map`/`HashMap` -> `*jrt.Map` (Java `hashCode`/`equals` semantics,
  so two cocoa `id`s with the same handle are one key), `List`/`ArrayList`/
  `ConcurrentLinkedQueue` -> `*jrt.List` (mutex-guarded). A call returning an erased type
  variable is wrapped in `jrt.Cast[T]` (nil stays nil). Any other unmapped `java.*` type is now
  `any` instead of an `unsupported_type_*` compile error - member calls on it are markers.
- **Exceptions**: `RuntimeException`/`Error`/`Exception`/`Throwable` as a *value* type are Go
  `error` (the catch dispatch already recovers them as `error`); `new Error()` ->
  `&jrt.JavaError{}`. `printStackTrace`/`getMessage` are intrinsics.
- **Native struct params**: a struct param whose Javadoc lacks `flags=struct` is passed by
  pointer (`OS.memmove(NSRect dest, ...)`, `objc_msgSendSuper(objc_super*)`,
  `_stret` results) - `*T` in the binding, `&x` (or a temp) at the call site. Before this, memmove
  wrote nowhere and `objc_msgSendSuper` got a struct where it wanted a pointer.
- **Constant natives**: `@method flags=const` or an Apple `kName` zero-arg native is a global
  read (`emitConstantAccessor`), not a call. `natives.properties` gained os.h's aliases
  (`objc_msgSend_bool`/`_fpret`/`_floatret` -> `objc_msgSend`, `objc_msgSendSuper_bool`).
- **Selector enum**: `Selector.valueOf(sel)` -> `sel`, a `Selector.sel_x` constant -> OS's own
  `sel_x` field, so `switch (Selector.valueOf(sel))` is a plain Go switch.
- **Switch expressions** anywhere (not only `x = switch ...`) hoist into a temp; `yield` values
  are adapted to the switch's type.
- **Casts, upcasts, instanceof are nil-safe helpers** generated per type pair:
  `cast<From>To<T>` (null -> nil, walks the impl cascade so `(NSWindow) new SWTWindow().alloc()`
  works, `ClassCastException` panic on mismatch), `upcast<From>To<T>` (`&x.Base` on a nil `x`
  panicked), `is<From>To<T>` (instanceof on null is false).
- **cocoa struct null**: a struct is its zero value when null (already true for assignment);
  `x == null` now compares against `T{}`. `isStruct` is limited to the cocoa package, so an swt
  data class (`DeviceData`) stays a nullable pointer.
- **Unresolved calls** no longer hoist their receiver/args into the prelude: they are referenced
  inside the panic closure, so `a && unresolved()` keeps short-circuiting.
- Smaller fixes: `'\0'`-style char literals emit as `'\u0000'`; a C-style `T name[]` field keeps
  its array type; a String field with no initializer is `""`; static initializers are
  numeric-adapted; a constructor whose Go name spells another class's `New<X>` gets a `_`
  suffix (`Device(DeviceData)` vs `new DeviceData()`); imports are filtered to those the body
  references; a try/synchronized closure that is a non-void method's last statement returns
  unconditionally (Go's "missing return"); an intrinsic that lowers to a bare value is `_ = x` as
  a statement.
- **JDK intrinsics added**: `Math.ceil/floor`, `Integer.valueOf/intValue`, `Boolean.
  booleanValue/parseBoolean/getBoolean`, `String.valueOf/indexOf/charAt/equalsIgnoreCase`,
  `Objects.requireNonNull/nonNull`, `System.nanoTime`, `System.out/err.println`,
  `System.getProperty(k, def)`, `Class.forName` (no-op), `Locale.getDefault().getLanguage()`
  (`$LANG`), `Runtime.version().feature()` (0: no JVM, so no AWT run-loop mode),
  `Cleaner.create` (nil), `Integer/Long/Float/Double.MAX/MIN_VALUE`, `Thread.MAX_PRIORITY`.
- **Manual (hand-written) additions**: `internal/cocoa/jni_manual.go` (`NewGlobalRef`/
  `JNIGetObject`/`DeleteGlobalRef` as a handle table - `Display.getWidget` maps a view's
  `SWT_OBJECT` ivar back through it), `internal/cocoa/nsexception_manual.go` (see next steps,
  item 3), `callback_manual.go` (above), `internal/jrt/{lang,util}.go`, `Display.isValidClass`
  (checks the Go package instead of a Java class name), `Thread.currentThread()` = goroutine id
  (so `checkWidget` really rejects other goroutines), `OS.setTheme`/`isSystemDarkAppearance`/
  `isAppDarkAppearance` are translated again (Display calls them).

## Round 7 api

Ergonomics for package `swt`'s public API, scoped to `swt` only (rule 5 of the round's own
brief) - `internal/cocoa`'s generated output is untouched (confirmed: `port.sh`'s second
invocation produces a byte-identical `internal/cocoa/*.go`).

### Upcast accessor + `<C>Like` interface (`TypeModel.assignLikeNames`, `ClassEmitter.emitLikeAccessor`)

Every non-struct, non-interface class `C` in package `swt` (in practice: every translated swt
class - `isStruct` is cocoa-only) gets:

```go
func (this *Composite) AsComposite() *Composite { return this }

type CompositeLike interface {
	AsComposite() *Composite
}
```

Go promotes `AsComposite()` through every embedded ancestor, so `*Shell`/`*Canvas`/`*Decorations`
satisfy `CompositeLike` too, for free, at any embedding depth - no per-class registration needed.
Computed once in `TypeModel.build` (`ClassInfo.asMethodName`/`likeInterfaceName`), so the
declaration site and every parameter-widening site (below) agree by construction.

**Collision check**: `asMethodName` suffixes `_` if a real Java method on that same class already
computes to `As<C>` (checked against `ci.declaredMethods`); `likeInterfaceName` suffixes `_` if
another swt-package class is itself already named `<C>Like`. Zero occurrences in the current file
set (confirmed: 0 `*Like_`/`As..._()` in the generated output).

### Parameter widening (`EmitUtil.publicParamList`)

A parameter of a widenable class type in an **exported, plain** signature becomes `<C>Like`; the
function's first lines convert it back to the concrete pointer:

```go
func NewButton(parentLike CompositeLike, style int32) *Button {
	var parent *Composite
	if parentLike != nil {
		parent = parentLike.AsComposite()
	}
	_ = parent
	...
```

(`_ = parent` only when nothing else in the body reads it - real SWT has genuine no-op overrides,
e.g. `Control.addRelation`, whose Java parameter is otherwise unused; Go rejects an unused local,
unlike an unused parameter.) A nil `parentLike` converts to a nil `*Composite` (`AsComposite` never
dereferences `this`), matching Java's own null-friendliness. Applied at exactly 3 emission sites:

- `ConstructorEmitter.emitConstructor` - only the public `New<X>` wrapper; `init<X>` (its callee,
  always unexported by shape) keeps `*C` - cheap, unchanged calls from every other generated site.
- `ClassEmitter.emitInstanceMethod` - only when the method is **plain**: not a cascade override
  (`ci.overridePoint(sig) == null`), not an implementation of a real Java `interface` method
  (`EmitUtil.implementsInterfaceMethod`, walks the type's transitive interfaces via
  `IMethodBinding.overrides`), and not the target of a `Type::method` reference anywhere in the
  file set (`TypeModel.isMethodReferenceTarget`, a one-time `ExpressionMethodReference` pre-scan
  over every compilation unit in `build()`). All three are real Go interface/function-value
  contracts shared with other code (the `<Root>Impl` cascade interface, a `ShellListener`-shaped
  interface, a `ListenerFunc`-adapted bound method) - widening only one side would break structural
  typing. `Item.handleDPIChange` (bound via `this::handleDPIChange`) and every `*Adapter`/interface
  implementation in `events/*` are the file set's real instances of this; excluded correctly.
- `ClassEmitter.emitStaticMethod` - static methods never participate in either contract, always
  widened.

Interface declarations themselves (`emitInterface`), the `<Root>Impl` cascade interface + its
default panic stubs, and `FunctionalEmitter`'s SAM-adapter/anonymous-class forwarder signatures
are untouched (still `emitter.paramList`, concrete `*C`) - by construction, since only the 3 call
sites above were switched to `EmitUtil.publicParamList`. Return types are never touched.

### SWT's static fields/methods drop the class prefix (`EmitUtil.staticFieldGoName`/`staticMethodGoName`)

`org.eclipse.swt.SWT` is a namespace of constants and static utility methods, not a real object -
`SWT.PUSH` -> `PUSH`, `SWT.error(int)` -> `Error`, not `SWTPUSH`/`SWTErrorFn`. Every other
translated class keeps its `<Class>Name` prefix, unchanged. One function each, called from both
the declaration site (`ClassEmitter.emitStaticFields`/`emitStaticMethod`) and every reference site
(`ExpressionEmitter.staticFieldRef`/`InvocationEmitter` via `Emitter.staticMethodGoName`), so a
declaration and its uses can never name-drift apart.

**Collision check**: the bare name falls back to the old prefixed form when it collides with
another translated class's `goTypeName` **or** a manual (hand-written) type's own same-Go-package
name (`Manual.ownPackageTypeNames`, new - a manual type living in another package, `jrt.*`, can't
collide). 2 real collisions in the current file set, both resolved this way:
- `SWT.Touch` (event-type constant, value 47) vs the manual `Touch` struct
  (`swt/widgets_stubs_manual.go`) -> stays `SWTTouch`.
- `SWT.LONG` (style-bit constant, `1<<28`) vs the manual `LONG` value type
  (`org.eclipse.swt.internal.LONG`) -> stays `SWTLONG`.

475 static fields and 9 static methods in `SWT.java`; 473/475 fields and all 9 methods got the
bare name, 0 needed the pre-existing type-collision `Fn` suffix (`SWT.error` no longer spells
`SWTError`, the real exception class, once the prefix is gone).

Hand-written callers updated for the new names: `swt/eventtable_test.go`,
`swt/widgets_widget_test.go`, `swt/layout_test.go`, `swt/widgets_stubs3_manual.go` (one
`SWTNONE` -> `NONE`).

### `cmd/hello`

```go
// before
shell := swt.NewShellDisplay(display)
shell.SetLayout(&swt.NewFillLayout().Layout)
button := swt.NewButton(&shell.Composite, swt.SWTPUSH)

// after
shell := swt.NewShellDisplay(display)
shell.SetLayout(swt.NewFillLayout())
button := swt.NewButton(shell, swt.PUSH)
```

Verified: `CGO_ENABLED=0 go build ./cmd/hello` and a 3s run, no crash, no stdout/stderr (same as
Round 6's baseline - a Screen-Recording-less `screencapture` couldn't confirm the window
on-screen then either, unchanged this round).

### Files changed

Translator: `TypeModel.java` (`asMethodName`/`likeInterfaceName` fields + `assignLikeNames`,
`methodReferenceTargets` + its pre-scan), `Manual.java` (`ownPackageTypeNames`),
`emit/EmitUtil.java` (`publicParamList`, `implementsInterfaceMethod`, `collidesWithTypeName`,
`staticFieldGoName`, `staticMethodGoName`, `SWT_NO_PREFIX_CLASS`), `emit/ClassEmitter.java`
(`emitLikeAccessor`, widened `emitInstanceMethod`/`emitStaticMethod`, delegates to the new
`EmitUtil` helpers), `emit/ConstructorEmitter.java` (widened `emitConstructor`),
`emit/ExpressionEmitter.java` (`staticFieldRef` delegates to `EmitUtil.staticFieldGoName`). All
components stay under the 450-line budget (`ClassEmitter` 304, `ConstructorEmitter` 204,
`EmitUtil` 124, `TypeModel` 299, `Manual` 249 - `Emitter.java` itself untouched, still 444).

Generated: all 67 `swt/*.go` files (regenerated - `internal/cocoa/*.go` byte-identical, 0 files
changed there). Hand-written: `cmd/hello/main.go`, the 3 test files + 1 manual file above.

End state green: `mvn -q -f tooling/j2go/pom.xml package && bash tooling/port.sh &&
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...` - same 33 tests, same 0 `internal/cocoa`
diff, same unsupported-marker counts as Round 6 (54 MethodInvocation, 6 ClassInstanceCreation, 5
CatchClause, 4 instanceof, 2 ExpressionMethodReference, 2 MethodDeclaration) - this round added no
new markers.

## Round 7 widgets

Translated for real: `Label`(565 lines)/`Menu`(1143)/`MenuItem`(1007)/`Text`(2522) (cocoa
widgets) and `GridLayout`(754)/`GridData`(577)/`FormAttachment`(320)/`FormLayout`(391)/
`FormData`(348) (common layout) - ~7,627 lines, all in `port.sh`'s single `swt` invocation
alongside Round 4-6's files (Display/Shell/Decorations/Control/Widget are regenerated in the same
pass, since Menu/MenuItem move here from the manual-stub list and every constructor/field access
on them needs to resolve to the real, overload-disambiguated names). Zero new unsupported markers
- the baseline count (54 MethodInvocation, 6 ClassInstanceCreation, 5 CatchClause, 4 instanceof,
2 ExpressionMethodReference, 2 MethodDeclaration) is unchanged from Round 6. `cmd/form` (hand-
written): a 2-column `GridLayout` form (`Label`+`Text` x2, a spanning `Button`), a `Menu`
`SWT.BAR`/`SWT.CASCADE`/`SWT.DROP_DOWN` File menu with a Quit item. Verified with an in-process
snapshot (`cacheDisplayInRect:toBitmapImageRep:`, same technique as Round 6): window titled
"Form", both text fields visible; `SetText` on the two `Text` fields followed by `performClick:`
on the OK button (the real target/action path, not a direct Go call) printed `Name: Alice` /
`Email: alice@example.com` through `Text.GetText()` reading the live `NSTextField`s back.

**Translator fixes, found by these files, all general (not per-file special cases):**

1. **`ConstructorEmitter.emitSuperInvocation`**: an explicit `super();` with *no* translated or
   manual superclass at all (`GridData`'s 5 constructors - a class that implicitly extends
   `java.lang.Object` can still write `super()`) crashed with a `NullPointerException` reading
   `ci.superclass.goFuncPrefix`. Now a real no-op (`return ""`), matching what Java's own
   `Object()` constructor does.
2. **`NumericEmitter` / `go vet`**: a ternary branch that assigns a variable to itself
   (`columnWidth = cond ? columnWidth : ...` - `GridLayout.computeSize`'s own column-width
   clamp) compiled but failed `go vet`'s self-assignment check. `emitCondIntoLvalue` now skips
   emitting a branch whose adapted text is textually identical to the lvalue.
3. **`InvocationEmitter.emitNew`**: `new String(char[], offset, count)` (`MenuItem`'s mnemonic-
   stripping code) only had the 1-arg `new String(char[])` shape implemented (README "Round 2").
   Now slices the buffer (`buf[off:off+count]`) before the same `utf16.Decode` call.
4. **`NumericEmitter.adaptNumeric`**: a manual int-backed enum field with no initializer
   (`Text.lastAppAppearance`, Java type `Display.APPEARANCE` - a nullable boxed enum) assigned or
   compared against `null` (`lastAppAppearance = null`) doesn't compile - Go's `Display_APPEARANCE`
   has no nil. Falls back to the type's zero value now, the same "known ceiling" `RoundingMode`'s
   zero value already documented: collapses "never set" into the enum's first constant instead of
   a distinct sentinel.
5. **`names.properties`**: `Label.createString()` (0-arg, renders the label's attributed text)
   shadows the promoted 7-arg `Control.createString(...)` it also calls - the exact same collision
   `Button.createString()` needed pinning for in Round 5. Renamed to `CreateAttributedText`.
6. **`StatementEmitter.emitWhile` - a real bug, found live, not by `go build`/`go vet`/`go test`**:
   Java's assignment-in-condition idiom (`while (widget == null && (view = view.superview()) !=
   null))`, `Display.LookupWidget`'s superview-walk) hoisted the assignment into a one-time
   prelude before a plain `for cond`, so `view` never advanced - an infinite loop, not a compile
   error. First hit at runtime: `cmd/form` hung forever inside `Label.CreateHandle`'s first
   `NSView.AddSubview` call, which synchronously re-enters Go through the window-proc callback
   before the new view is registered, sending `DisplayLookupWidget` walking a superview chain
   that never finds a match - confirmed by a `SIGQUIT` goroutine dump (`objc_msgSend`/
   `object_getInstanceVariable` on the stack, called over and over). Fixed generally: `emitWhile`
   now probes the condition's prelude into a throwaway buffer first: empty prelude keeps the
   existing `for cond` shape unchanged, a non-empty one (a Java condition with a real side effect)
   unrolls into `for { <prelude>; if !(cond) { break }; <body> }` so the side effect re-runs every
   iteration, matching Java's own re-evaluation semantics. Regenerating the whole `swt` package
   with this fix touched only `Display.LookupWidget` - no other `while` in the translated set has
   a side-effecting condition.

**`Manual.java`/`manual.txt`**: `Menu`/`MenuItem` removed from `ENTRIES` and from `WIDGET_SUPER`
(both real `ClassInfo`s now); their opaque stub definitions removed from
`swt/widgets_stubs2_manual.go`. One new stub needed: `TrayItem.ShowMenu` (a no-op - `Menu._setVisible`
calls it for a tray popup menu, and `Tray`/`TrayItem` themselves are still out of scope).

**JUnit layout tests ported** (`swt/layout_test.go`): `Test_org_eclipse_swt_layout_GridData` (pure
field/constant checks) and `Test_org_eclipse_swt_layout_FormAttachment` (the Control-taking
constructors only ever store the pointer, never dereference it, so a bare zero-value `*Shell`
stands in for the original test's live one - no `Display` needed). No `Test_org_eclipse_swt_
layout_GridLayout.java`/`FormLayout`/`FormData` exists upstream to port.

**A live `Display` cannot be created inside `go test` on this machine - confirmed empirically, two
independent failure modes**: (1) a plain `Test*` function runs on its own goroutine (`go
tRunner(...)`), never the process's real thread 1, regardless of its own `runtime.LockOSThread()`
- `Display.create` panics with "Display must be created on main thread due to Cocoa restrictions"
(`SWT.error`, real SWT's own check, not a gowt shortcut). (2) creating the `Display` inside
`TestMain` (which *does* run on the real main goroutine) doesn't help either: SWT's own thread-
affinity check (`checkWidget`, Round 6's goroutine-id-based `Thread.currentThread()`) compares the
creating goroutine against the calling one, and a `Test*` function is always a different goroutine
from `TestMain` - `Shell.create` panics the same way from there. A `GridLayout.computeSize` test
with real controls is therefore not attempted; `layout_test.go` stays pure-value/no-live-view, per
the precedent Round 5 already set for `FillLayout`/`RowLayout`.

## Round 7 gfx: GC and the graphics resources, the paint path

**Status: done, green.** `mvn -q -f tooling/j2go/pom.xml package && bash tooling/port.sh &&
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...` pass. `cmd/paint` (hand-written): a
Shell with a Canvas (white background) whose PaintListener fills a rectangle, draws a 3px line,
an oval and a string. The real path runs: `drawRect:` -> `OS.CALLBACK_drawRect_` (NSRect by
value) -> `Display.windowProc` -> `Canvas.drawRect` -> `Widget.drawRect` -> `Control.drawWidget`
-> `SWT.Paint` with a `GC` from `GC.cocoa_new`. Checked with an in-process
`cacheDisplayInRect:` snapshot (scratch program); `Path` fill/draw, `drawText` with
`DRAW_TRANSPARENT` and `TextLayout.draw` render too. `cmd/hello` still runs.

**Translated for real** (swt invocation in `port.sh`): `GC`, `GCData`, `Drawable`, `FontMetrics`,
`LineAttributes`, `Pattern`, `Transform`, `Path`, `PathData`, `Region`, `Image`, `ImageData`,
`PaletteData`, `ImageDataProvider`, `ImageFileNameProvider`, `ImageDataAtSizeProvider`,
`ImageGcDrawer`, `Cursor`, `TextLayout`, `TextStyle`, `GlyphMetrics`. Their stubs are gone.

**Manual** (manual.txt): `GC.GCTextData.Key`/`Cache` (`swt/graphics_gc_manual.go`: a record key
and an insertion-ordered cache that releases the evicted layout at `size >= cacheSize`, as
`removeEldestEntry` does); `ImageDataLoader`, `FileFormat.DEFAULT_ZOOM`, `ImageColorTransformer`
(default "grayed" transform), `StrictChecks`, and the `DPIUtil` scaling functions
(`swt/graphics_stubs_manual.go`). Image loading from files/streams is not ported:
`ImageDataLoader.load*` panics, `canLoadAtZoom`/`isDynamicallySizable` are false.

**Contract changes** (general rules):

- **Hex literals**: `stripNumericSuffix` stripped `f`/`d` from hex tokens (`0xFF` -> `0xF`). This
  silently broke existing output too (`SWT.DEL` was `0x7`, `SWT.KEY_MASK` `0xFFF`, `Color`'s
  `alpha &= 0xF`). Hex tokens now lose only `l`/`L`. A hex `int` literal above `0x7fffffff` is
  emitted as its negative decimal (`0xFF000000` -> `-16777216`), a negative hex `long` likewise.
- **Narrowing constant casts**: a cast that JDT folds to a constant (`(byte)0xFC`) emits the
  Java-truncated value (`int8(-4)`); Go rejects overflowing constant conversions.
- **Binary numeric promotion**: a `byte`/`short`/`char` operand of a binary operator is converted
  to `int32` (`NumericEmitter.adaptBinaryOperands`), as Java promotes it. Before, `b & 0xFF` on a
  byte stayed `int8` (compile error or wrong result).
- **`>>>`** (and `>>>=`, also inline): `int32(uint32(x) >> n)` / `int64(uint64(x) >> n)`.
- **Inline compound assignment** (`sp = spr += d`): applied in place, its value is the new lhs.
  It used to drop the operator (`spr = d`).
- **Loop conditions and updaters with hoisted statements**: a `for`/`while` condition that needs
  a prelude (`(bit >>= b) != 0`, `--index >= 0`) is re-evaluated at the top of each iteration
  (`for { prelude; if !(cond) { break } ... }`); it used to run once before the loop (an
  infinite loop in `ImageData`'s static init; `Synchronizer`'s `while (--index >= 0)` was also
  affected). The general `for` form puts its updaters in a post-statement closure
  (`for ; cond; func() {...}() {`), so `continue` still runs them, and is braced so its hoisted
  init vars keep loop scope.
- **Interface default methods**: the body becomes `<Iface>Default<M>(this <Iface>, ...)`; every
  implementer that doesn't declare the method gets a forwarder (`Emitter.defaultForwarders`:
  translated classes, anonymous classes, `<Iface>Func` adapters), so the Go interface lists
  default methods like abstract ones. `Control`/`Device`/`Image` get `IsAutoScalable` this way.
- **Inner (non-static member) classes**: an `this_0 *Outer` field; `new Inner()` sets it after
  construction (prelude), and an unqualified outer member resolves to `this.this_0`
  (`Emitter.implicitThis`). Ceiling: the inner constructor itself can't reach the outer
  instance (none does in the translated set).
- **Field names that are Go keywords** (`GCTextData.range`) get a `_` suffix
  (`EmitUtil.fieldIdent`).
- **Catch of an unmapped JDK exception** (`IOException`, `NumberFormatException`) is a dead
  clause (`if false { var e error ... }`); it used to be `case any:`, which caught every panic.
- **`super.equals`/`super.hashCode` resolving to `Object`**: identity (`any(this) == o`,
  `jrt.IdentityHashCode`).
- **Casts of a lambda/method reference** to its functional interface emit the value as is.
- **try-with-resources on an `any`-typed resource** closes it only if it has `Close()`.
- **Unresolved `new X(args)`** references its args inside the panic closure (as unresolved calls
  already did), so a local used only there doesn't go "declared and not used".
- `stmtExits` treats `if/else` with both branches exiting as exiting (no unreachable
  `fallthrough`).
- **JDK intrinsics**: `Math.round` (floor(x+0.5), int for float / long for double),
  `Math.hypot`, `Float.floatToIntBits`, `new String(char[], offset, count)`.

**Markers left** (swt invocation): MethodInvocation 130, ClassInstanceCreation 14,
CatchClause 7, ExpressionMethodReference 2, MethodDeclaration 2. New ones from this round are
JDK surface off the paint path: `StringBuilder` (`TextStyle.toString`, `TextLayout`'s tab
expansion), `Optional`/`ByteArrayInputStream`/`DPIUtil.ElementAtZoom` (`Image` from providers
and streams), `FontMetrics.equals`'s `Double.compare`, `Objects.hash`.

**Open gaps**: image loading (ImageLoader + codecs); `Image` from `ImageFileNameProvider`/
`ImageDataProvider` at non-100 zoom goes through unported `ElementAtZoom`/`Optional` calls;
`DPIUtil` is a partial hand port (deviceZoom = native zoom, no `swt.autoScale`); records and
`LinkedHashMap` subclasses have no translator rule (GC's cache is hand-written); Java's shift
count masking (`& 31`) is not emitted for `<<`/`>>`/`>>>`.

## Round 8 tree

Translated for real (swt invocation in `port.sh`): `Tree` (3560 Java lines), `TreeItem`, `TreeColumn`,
`ScrollBar`. The `ScrollBar` stub is gone from `swt/widgets_stubs2_manual.go` and from `Manual`
(`ENTRIES`, `WIDGET_SUPER` is now empty). Everything else they need was already translated. No new
unsupported markers (same 130/14/7/2/2 as Round 7 gfx).

The NSOutlineView data source/delegate selectors (`outlineView:numberOfChildrenOfItem:`,
`child:ofItem:`, `isItemExpandable:`, `objectValueForTableColumn:byItem:`, `willDisplayCell:...`,
`outlineViewSelectionDidChange:`, `shouldExpandItem:`, `expandItem:expandChildren:`, ...) and the
`SWTImageTextCell` drawing methods were already registered by `Display.initClasses` since Round 6;
the struct-by-value ones (`highlightSelectionInClipRect:`, `drawBackgroundInClipRect:`,
`drawInteriorWithFrame:inView:`, `canDragRowsWithIndexes:atPoint:`, `imageRectForBounds:`, `cellSize`)
go through the generated `OS.CALLBACK_x` IMPs. No bridge change was needed.

`cmd/tree` (hand-written): Shell + FillLayout + `Tree(BORDER)`, three roots with 2-3 children,
"Fruits" expanded via `SetExpanded`, Selection and Expand listeners. `Event.Item` is a `*Widget`, so
the program maps `item.AsWidget()` back to its `*TreeItem`. Checked with an in-process snapshot:
`selectRowIndexes:byExtendingSelection:` printed `Selected: Banana`, `expandItem:` printed
`Expanded: Grains`, a synthesized down-arrow `keyDown:` moved the selection to `Cherry` and printed it.

**Contract changes** (general rules):

- **Cascade collision check by Go name** (`TypeModel.build`): a cascade method gets the
  `On<Class>` suffix only when another signature in its tree computes the same Go name, not merely
  the same Java name. Before, `TreeItem.getBounds(int)` (Go `GetBoundsIndex`) pushed
  `Control.getBounds()` to `GetBoundsOnControl`, and `Item.setText(String)` to `SetTextOnItem`.
  Side effect: many existing names lost a spurious suffix, public and internal. Renames visible to
  hand-written code: `Control.SetBackgroundColor_196` -> `SetBackgroundColor` (the NSColor one is
  now `SetBackgroundColorOnControl`), `Layout.LayoutOnLayout` -> `LayoutFn`,
  `ComputeSizeOnLayout` -> `ComputeSize`, `Device.GetBoundsOnDevice` -> `GetBounds`, and the
  internal `...OnControl`/`...OnWidget` forms (`SetZOrder`, `SendKeyEvent`, `IsOpaque`, ...).
  Callers in `cmd/paint` updated.
- **names.properties**: `TreeItem.setText(String[])` -> `SetTexts`, `setImage(Image[])` ->
  `SetImages`. They are declared before the `Item.setText(String)`/`setImage(Image)` overrides and
  would otherwise take the bare name, pushing `Shell.SetText` onto a suffixed cascade name.
- **Array args of natives with more than 8 integer args** (`NativeEmitter.emitLazyNative`):
  purego's darwin/arm64 stack packing copies a slice argument's whole header (ptr, len, cap), not
  its pointer, so an array landing on the stack arrived as its length. The backing func takes
  `*elem` and the wrapper passes `unsafe.SliceData(x)`. Only `OS.UCKeyTranslate` has that shape;
  before the fix any key press on any control crashed (`SIGSEGV addr=0x1` in `UCKeyTranslate`,
  via `Widget.calculateKeycode`).

**Gaps**: `Tree.deselectAll()` is `DeselectAll0` (Widget's `deselectAll(id,sel,sender)` cascade takes
the bare name); mouse clicks, collapse, columns, images, `VIRTUAL`/`CHECK` and scrolling
were not exercised live.

## Round 8 stack

**Status: done, green.** `mvn -q -f tooling/j2go/pom.xml package && bash tooling/port.sh &&
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...` all pass.

**`org.eclipse.swt.custom` package placement**: no change needed. `Main.goPackageDir` already
maps every non-`org.eclipse.swt.internal.cocoa` Java package to Go package `swt`
(`GoTypes.isCocoaPackage`), so `org.eclipse.swt.custom` lands in `swt` for free. Checked for
cycles and collisions before translating: nothing under `widgets`/`graphics` imports
`org.eclipse.swt.custom` (one-directional, `custom` -> `widgets`/`graphics` only), and grepping
the generated `swt`/`internal/cocoa` output for `Group`/`Sash`/`SashForm`/`StackLayout` found no
existing Go symbol of those names - zero collisions. Only change: one new entry in `Main.java`'s
`SOURCE_ROOTS` (`bundles/org.eclipse.swt/Eclipse SWT Custom Widgets/common`, where
`StackLayout.java`/`SashForm.java`/`SashFormLayout.java`/`SashFormData.java` live) and 6 new file
arguments added to `port.sh`'s single `swt` invocation, right after `Text.java`. No translator
code changed at all this round.

**Translated for real**: `Group`, `Sash` (cocoa widgets - both `widgets_group.go`/`widgets_sash.go`),
`StackLayout`, `SashFormLayout`, `SashFormData`, `SashForm` (`custom_*.go`). Every dependency these
six files need - `NSBox`/`SWTBox` (Group's native handle), `Display.boxFont`, the accessibility
bridge (`Accessible`/`ACC.CHILDID_SELF`, `NSMutableArray`, the `NSAccessibility*` string constants
Sash reads for its role/orientation/value attributes), `Control.FixMnemonic`, `this::method`
references (`SashForm`'s `sashListener = this::onDragSash`), `System.arraycopy`, layout-data
`instanceof` casts (`SashFormData`) - was already generated by earlier rounds (Composite/Canvas
already exercise the same accessibility path; Display already wires `boxFont` off `NSBox`'s title
font). Zero new unsupported markers: the swt-invocation baseline (130 MethodInvocation, 14
ClassInstanceCreation, 7 CatchClause, 2 ExpressionMethodReference, 2 MethodDeclaration, unchanged
since Round 7 gfx) is unchanged by this round's 6 files.

**Fallout, not a bug**: adding `SashForm.java` - which overrides `Composite.setLayout(Layout)` and
`Control.setBackground(Color)`/`setForeground(Color)` for the first time in the translated set -
moved those three methods into the `<Root>Impl` override cascade (README "Round 7 api"'s widening
rule correctly stops widening a cascade method's parameter, since only one side of a shared
interface can change). `Composite.SetLayout` reverted from accepting `LayoutLike` back to its
concrete `*Layout` parameter (matching the pre-Round-7-api shape), and `Control`'s public
`setBackground(Color)`/`setForeground(Color)` got new cascade-disambiguated Go names,
`SetBackgroundColorOnControl`/`SetForegroundOnControl` (were `SetBackgroundColor_196` - an
index-suffixed name from the older cross-family-collision path, README
"resolveCrossFamilyNameCollisions" - and plain `SetForeground` respectively; both now go through
the `base + "On" + declaringClass` cascade-name path instead, since the method is a real override
point as of this round). Regenerating the whole `swt` package
also renumbered every anonymous-class/temp-var counter (`anon218` -> `anon232`, `cond351` ->
`cond365`, ...) purely because the 6 new files sit earlier in `port.sh`'s file list than
`Resource.java` onward - confirmed with `git diff --stat` before touching anything: every changed
line in files other than the 6 new ones is one of these two purely mechanical renamings, nothing
semantic. Fixed the 3 hand-written commands that broke: `shell.SetLayout(swt.NewFillLayout())` ->
`shell.SetLayout(&swt.NewFillLayout().Layout)` (`cmd/hello`, `cmd/paint`, and the `GridLayout`
equivalent in `cmd/form`), `canvas.SetBackgroundColor_196(...)` ->
`canvas.SetBackgroundColorOnControl(...)` (`cmd/paint`).

**`cmd/stack`** (hand-written, new): `Shell` with a `SashForm` (`SWT.HORIZONTAL`) - left a
`Composite` with 3 `Button`s "A"/"B"/"C", right a `Composite` with a `StackLayout` holding 3
`Group`s ("Panel A/B/C", each with a `Label`). Each button's `SelectionListener` sets
`stackLayout.TopControl` and calls `right.Layout()`. Verified with an in-process
`cacheDisplayInRect:` snapshot (scratch program, not in the repo): window titled "Stack", the sash
splits the two sides (`SetWeights([]int32{1, 3})`), clicking button B via `performClick:` on the
real `NSButton` (the target/action path, not a direct Go call) printed "B clicked" through the
real listener and switched the visible panel to "Panel B" / "This is panel B" - `Group`'s box
title and `StackLayout`'s visibility toggle both confirmed on screen.

**Not attempted**: nothing - Sash/SashForm went through cleanly, no new translator work needed.

## Round 8 merge: cascade Go-name collision (wt/tree x wt/stack)

Merging wt/stack (this round, above) into wt/tree (Round 8 tree's "Go-name, not Java-name"
cascade check) surfaced a case neither round's fix alone covered: two members of the *same*
cascade, declared on the *same* class, computing the *same* pre-suffix Go name. `Control`
declares `setBackground(Color color)` (not the first-declared `setBackground` overload, so it
gets the overload suffix `+Color` -> `SetBackgroundColor`) and, separately, `setBackgroundColor
(NSColor nsColor)` (its own bare name, first-declared in its own family -> also
`SetBackgroundColor`). Neither was overridden below `Control` before this round, so both stayed
non-cascade and the coincidence was harmless. `SashForm` (this round) overrides `setBackground
(Color)` for the first time - now both are cascade members of the same tree, `TypeModel.build`'s
`base + "On" + declaringClass` suffix computes `SetBackgroundColorOnControl` for *both*
(same base, same declaring class), and `go build` fails with "duplicate method".

`TypeModel.build`'s cascade-naming pass now groups a root's cascade methods by their pre-suffix
base name first (`assignCollidingCascadeNames`): the member whose *bare* Java name literally is
that base (here `setBackgroundColor` - no overload suffix involved) claims the plain
`<base>On<class>` spelling; any other member that still collides after the class suffix appends
its own parameter types, then a stable ordinal if even that repeats. Only a colliding group is
touched - a lone cascade member's name is unchanged, so this is not a per-method pin.

Names visible to hand-written code: `Control.setBackground(Color)` is now
`SetBackgroundColorOnControlColor` (was going to collide as `SetBackgroundColorOnControl`);
`Control.setBackgroundColor(NSColor)` keeps `SetBackgroundColorOnControl` from this round's
earlier fix. `cmd/paint`'s `canvas.SetBackgroundColor(...)` updated to
`canvas.SetBackgroundColorOnControlColor(...)`; `cmd/paint` and `cmd/tree` also still had
`shell.SetLayout(swt.NewFillLayout())` from before the `SetLayout` widening was lost (see below) -
both updated to `shell.SetLayout(&swt.NewFillLayout().Layout)`.

**Known gap, not fixed this round**: `Composite.SetLayout` lost its `LayoutLike` widening
("Round 7 api") the moment `setLayout(Layout)` became a cascade method - a cascade method's own
parameter can't be widened because only one side of the shared `WidgetImpl` interface may change,
so every caller now passes a concrete `*Layout` (`&swt.NewFillLayout().Layout`) instead of a
`LayoutLike`. "Round 7 api"'s widening rule could still apply to a cascade method by splitting the
two roles: the cascade/override point keeps the narrow, unexported, impl-interface signature
(`setLayoutImpl(*Layout)` or similar, whatever `x.impl.M(...)` dispatches through), while each
concrete type also gets a small exported wrapper (`SetLayout(l LayoutLike)`) that narrows via the
existing `As<GoTypeName>`/cast helper and calls the impl method - the same shape the `Like`
interfaces already give non-cascade methods, just moved one level down so the wrapper, not the
interface method itself, does the widening.

## Round 9 api: public API independent of the translated set

Before this round a method's exported Go name and signature depended on whether some translated
subclass overrode it: a cascade method lost the "Round 7 api" `CLike` widening and could get
`On<Class>`/param-type suffixes (adding `SashForm` turned `Composite.SetLayout(LayoutLike)` into
`SetLayout(*Layout)` and produced `SetBackgroundColorOnControlColor`). Three rules fix that; they
apply to classes whose cascade root is in package `swt` (`ClassInfo.splitsDispatch`).
`internal/cocoa` keeps the old exported cascade names - it is reached from `swt` through
`.Impl()`, and an unexported dispatch name would not be callable across packages. Its output is
byte-identical.

**1. Dispatch and public surface are separate** (`TypeModel.putCascadeName`,
`ClassEmitter.emitDispatchWrapper`). A cascade method's dispatch name is its natural Go name
decapitalized plus `_` (`setLayout_`, `layoutFn_`, `init_`). The `<Root>Impl` interface, the
root's default panic stubs, every override in a subclass, anonymous subclasses, `super.x()` and
every generated call through `.impl` use that name with the concrete `*C` signature. At the
override point (the topmost declaring class) the exported natural-name method is a wrapper with
the "Round 7 api" widened params: it converts once and calls `this.impl.<dispatch>(...)`.
Subclasses get no wrapper of their own: the point's wrapper is promoted, and `.impl` reaches the
most-derived override. An abstract method at the point (`Layout.computeSize`) gets only the
wrapper. A non-cascade method is emitted as before: natural name, widened params, the body
directly. Java-interface implementations and `Type::method` targets keep narrow params in the
wrapper too (same reason as in "Round 7 api").

```go
func (this *Composite) SetLayout(layoutLike LayoutLike) {
	var layout *Layout
	if layoutLike != nil {
		layout = layoutLike.AsLayout()
	}
	this.impl.setLayout_(layout)
}

func (this *Composite) setLayout_(layout *Layout) {
	this.CheckWidget()
	this.layout = layout
}

func (this *SashForm) setLayout_(layout *Layout) { ... } // override, reached through Widget.impl
```

The exported name comes only from `Names.goMemberName` for the declaring class's own method, so
it cannot change when a subclass is translated. The root's default panic stubs are unexported
now, so they no longer show up as callable methods on every widget (`Button.SetLayout` used to
panic).

**2. Overload suffix never equals another method's natural name** (`Names.isNaturalNameOfOther`).
If `base + CapitalizedParamNames` equals the unsuffixed Go name of a differently named Java
method of the same kind, the name becomes `base + "With" + CapitalizedParamNames`. Same kind
means instance methods declared in the class or its superclasses below `java.lang.Object`, or
statics of the same class. The check uses JDT bindings, so reference-only and untranslated
classes count too. `Control.setBackground(Color)` -> `SetBackgroundWithColor`, because
`Control.setBackgroundColor(NSColor)` is `SetBackgroundColor`. `names.properties` pins still win.
`TypeModel.resolveCrossFamilyNameCollisions` stays as a safety net for suffix-vs-suffix
collisions. In `swt` it no longer lets cascade names claim first.

**Type-name guard** (`Names.withTypeNameGuard`): an instance method whose Go name equals its own
class's or an ancestor's Go type name gets `Fn`, since that name is also the embedded field of
every subclass. `Layout.layout()` stays `LayoutFn`, as it was via the cascade rule. New:
`Transform.transform(float[])` -> `TransformFn`.

**3. Collision rules only touch dispatch names.** `On<Class>`/param-type tags
(`assignCollidingCascadeNames`) now apply only to dispatch names, and only among one root's
cascade members: a dispatch name is lowercase with `_`, so it cannot collide with a
non-cascade method or an embedded type name. `InvocationEmitter`'s manual-receiver path (the
hand-written `Caret`/`IME` stubs) calls the natural name, since a stub mirrors the public API.

**Stability check** (`tooling/apidump`): prints the exported API of package `swt` (package-level
funcs/vars/consts/types plus every exported type's full method set, promoted methods included,
via `go/types`), sorted, one symbol per line:

```sh
CGO_ENABLED=0 go run ./tooling/apidump > /tmp/api.txt
```

Checked: dump, remove `SashForm`/`SashFormLayout`/`SashFormData` from `port.sh` (and their
generated files), regenerate, dump again, diff. The only differences are the 568 removed `SashForm*`
lines: types `SashForm`/`SashFormData`/`SashFormLayout`, their `*Like` interfaces, methods,
`NewSashForm`, `SashFormCheckStyle`, `SashFormDRAG_MINIMUM`. Nothing on `Composite`, `Control`,
`Widget` or any other type changed.

**Renames visible to hand-written code** (vs the Round 8 merge): `SetBackgroundColorOnControlColor`
-> `SetBackgroundWithColor`, `SetBackgroundColorOnControl` -> `SetBackgroundColor` (NSColor),
`SetForegroundOnControl` -> `SetForeground`, `SetFontOnControl` -> `SetFont`,
`SetOrientationOnControl`/`OnWidget` -> `SetOrientation`, `SetImageOnItem`/`OnWidget` ->
`SetImage`, `DrawBackgroundOnWidget` -> `DrawBackground`, `InitOnResource` -> `Init`,
`Transform.Transform` -> `TransformFn`. Cascade methods take `CLike` again
(`shell.SetLayout(swt.NewFillLayout())`). `cmd/*` and `swt/graphics_test.go` (`p.impl.Clone()`
-> `p.Clone()`) are updated.

**Known ceiling**: a subclass method that is a Java overload (not an override) of an inherited
method takes its own class's first-overload name and shadows the inherited one on that subclass.
For example, `Composite.drawBackground(GC,...)` -> `DrawBackground` hides
`Widget.DrawBackground(id, context, rect)` on Composite and below. This is stable, since it
depends only on the ancestors, but those inherited overloads are reachable only through the
embedded field (`c.Widget.DrawBackground`). The fix would be to count inherited same-name methods
in the overload order.

## Round 9 widgets

**Status: done, green.** `mvn -q -f tooling/j2go/pom.xml package && bash tooling/port.sh &&
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...` all pass. `internal/cocoa` content
unchanged (still byte-identical, same 210 PI/cocoa files + `C.java`). `swt`'s invocation baseline
unsupported-marker count (130 MethodInvocation, 14 ClassInstanceCreation, 7 CatchClause, 2
ExpressionMethodReference, 2 MethodDeclaration - unchanged since Round 7 gfx) is unchanged by this
round's 16 new files - zero new markers.

**Translated for real** (`port.sh`, right after `TreeColumn.java`): `Dialog` (common), `ColorDialog`/
`FontDialog`/`MessageBox`/`TabItem`/`TabFolder`/`Combo`/`Table`/`TableItem`/`TableColumn` (cocoa
widgets), `ScrolledCompositeLayout`/`ScrolledComposite`/`ControlEditor`/`TableEditor` (custom,
`Eclipse SWT Custom Widgets/common` - already on `Main`'s sourcepath since Round 8 stack). ~8,750
generated lines. `Table` reuses `Tree`'s NSTableView/NSOutlineView callback machinery (registered
by `Display.initClasses` since Round 6) - no bridge change needed, same as Round 8 tree's own note
about `Tree`.

**Manual stubs retired**: `ColorDialog`/`FontDialog`/`Combo` (were opaque callback-receiver shims in
`swt/widgets_stubs3_manual.go`, only ever holding the selector methods `Display.dialogProc`/
`Display.applicationProc` already cast to and called by Go name - `Display.java` was translated
back in Round 6 already assuming these types were real, so the real translation's method names
matched the manual shim's exactly, no `Display` changes needed) and `Dialog` (was an empty
`struct{}` in `swt/widgets_stubs_manual.go`, only ever compared to `nil`). Removed from both
`tooling/j2go/manual.txt` (documentation copy) and the actual source of truth,
`Manual.java`'s `ENTRIES`/`DIALOG` constant and the `widgets.*` stub-registration array - **manual.txt
is not read at runtime** (confirmed by grep: nothing outside comments references the file), so a
manual-type retirement needs both files edited, or the translator crashes with a null `ClassInfo`
lookup at the newly-freed class (`ClassEmitter.emitClass`, `ci == null`) - hit and fixed this round.
`FileDialog`/`TaskBar`/`TaskItem`/`Tray`/`TrayItem` stay manual (out of this round's scope).

**Translator fixes, found by these files, both general (not per-file special cases):**

1. **`names.properties`**: `Combo.createString(String)` (renders an item/text attributed string)
   shadows the promoted 7-arg `Control.createString(...)` it also calls - the same collision
   `Button`/`Label`'s own zero-arg `createString()` needed pinning for in Rounds 5/7, just with a
   1-arg erasure key this time (`Combo#createString(java.lang.String)=CreateAttributedString`).
2. **`names.properties`**: `TableItem.setText(String[])`/`setImage(Image[])` (declared before the
   `Item.setText(String)`/`setImage(Image)` overrides they also have) shadow the bare cascade name
   as `TreeItem`'s own array overloads did in Round 8 tree - pinned the same way
   (`TableItem#setText([Ljava.lang.String;)=SetTexts`, `#setImage([Lorg.eclipse.swt.graphics.Image;)=SetImages`).
   Without this, `go build` renamed the whole `Item` cascade's `setText(String)` from bare `SetText`
   to `SetTextOnItem` again (same symptom as Round 8's `TreeItem` fix, now proven general: *any*
   sibling class in the tree with an unpinned array overload pushes the same rename, not just
   `TreeItem`).

**`cmd/widgets2`** (hand-written, new): a `Shell` with a `TabFolder` of 3 tabs - (1) a `RowLayout`
`Composite` with a `READ_ONLY` `Combo` (3 items) and an editable `Combo` (3 items), each with a
`SelectionListener`; (2) a `Table` (`BORDER|FULL_SELECTION`, 2 columns with headers visible, 5
rows, one `SelectionListener`); (3) a `ScrolledComposite` over a tall `RowLayout` `Composite` of 30
`Label`s. Verified with an in-process `cacheDisplayInRect:` snapshot (scratch program, not in the
repo, module `snapwidgets2` replacing `github.com/haiodo/gowt` with this worktree): all three tabs
render correctly after `tabFolder.SetSelectionIndex(n)` (Combo tab: both combos visible with their
first items; Table tab: 2 headered columns, 5 rows; Scrolled tab: `Label 1`.."Label 16" visible,
clipped by the viewport, confirming the content (573px tall) is taller than the visible area (302px)
and scrolls). Selection fired through the real native paths, not direct Go calls: the read-only
combo (`NSPopUpButton`) via `selectItemAtIndex:` + `sendAction:to:` (`sendAction:to:` is what
`NSControl` itself calls after a real click - `NSPopUpButton` has no `performClick:`-with-index
equivalent, unlike `NSButton`); the editable combo (`NSComboBox`) and the table row via
`selectItemAtIndex:`/`selectRowIndexes:byExtendingSelection:` alone, which - like `NSOutlineView` in
Round 8 tree - post their selection-changed notification on any change regardless of source. All
three listeners printed the expected text (`Combo (read-only) selected: Three`, `Combo (editable)
selected: Beta`, `Table selected: Row 3`), matching the item/row actually selected (index 2, index
1, index 2).

**Not a translator bug, a usage bug caught and fixed**: the first `cmd/widgets2` draft called only
`scrolled.SetExpandHorizontal(true)` (not `SetExpandVertical`) and relied on `SetMinSize` alone for
the content's height. Real `ScrolledCompositeLayout.layout()` (`Eclipse SWT Custom Widgets/common`)
only overwrites `contentRect.height` from `minHeight` when `expandVertical` is true; with it false,
`contentRect` starts from `content.getBounds()` - the content's *current* bounds, which stay
`{0,0,0,0}` forever if nothing ever calls `content.setSize(...)` directly. `minWidth`/`minHeight` by
themselves only feed the scrollbar-range math and the layout's own `computeSize`, not the actual
content resize when not expanding. Fixed by also calling `SetExpandVertical(true)` (the standard
real-SWT idiom for a content composite taller than its viewport: expand both axes, then
`SetMinSize` clamps the *smaller* dimension up when the viewport is bigger than the content) -
matches upstream `ScrolledComposite` snippets, not a gap in the translated code itself.

**Gaps**: `FileDialog` stays a manual stub (not in this round's file list). `ColorDialog`/
`FontDialog`'s modal `open()` (`NSColorPanel`/`NSFontPanel` + `runModalForWindow:`) was not run live
- constructing `ColorDialog`/`FontDialog` and building doesn't crash, but the blocking modal loop
itself is unverified (matches the task's own "runtime check optional (they block)"). `MessageBox`'s
sheet-mode path (`style & SWT.SHEET`) constructs a real `Callback` (`new Callback(this,
"_completionHandler", 1)`), which is still the manual reflection-stub that panics on `GetAddress`
(`swt/widgets_stubs3_manual.go`, unchanged) - only the non-sheet `runModal()` path is safe to
exercise; not run live either. `TableEditor`/`ControlEditor` translated cleanly (no unsupported
markers) but have no `cmd/*` exercising them yet - `ControlExample`'s own `Tab.java` does not
reference either class directly (checked by grep), so nothing in this round's brief required
driving them live. Mouse-driven column resize/reorder, `Table`'s `CHECK`/`VIRTUAL` styles, and
`Combo`'s autocomplete/verify-text paths were not exercised live.

## Round 9 images: load images from files and streams

**Status: done, green.** `mvn -q -f tooling/j2go/pom.xml package && bash tooling/port.sh &&
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...` all pass. `cmd/images` (hand-
written): a Shell with a Canvas whose PaintListener draws 3 images (PNG, GIF, BMP) loaded via
`getResourceAsStream -> ImageData(InputStream) -> Image(Display, ImageData)`, through
`gc.DrawImage`. Checked with the same in-process `cacheDisplayInRect:toBitmapImageRep:` snapshot
`cmd/paint` uses: PNG "SWT" text (translucent pink on white), GIF folder icon (correct colors,
transparent background), BMP red square all render correctly. `cmd/hello`/`paint`/`form`/`tree`/
`stack` still run.

**Direction change mid-round**: the first pass translated `FileFormat`'s dispatch as a manual
type plus a full j2go port of `PNGFileFormat`/`GIFFileFormat`/`WinBMPFileFormat`/`OS2BMPFileFormat`/
`LZWCodec`/`LZWNode`/`Png*`/`LEDataInputStream`/`LEDataOutputStream` (~1900 Java lines across 20
files) - it translated cleanly and a PNG decoded correctly through it (zlib inflate via Go's
`compress/zlib`, PNG's own dead "3.2" hand-rolled-inflate fallback and its `PngEncoder`/save path
dropped as out of scope). Redirected before finishing GIF/BMP: less code and a proven decoder set
(Go's stdlib png/gif/jpeg plus `golang.org/x/image/bmp`) beats re-verifying a translated codec's
every color-type/interlace/compression branch against a decoder nothing else exercises. `Image
Loader`/`ImageData`/`ImageLoaderEvent`/`ImageLoaderListener` still translate for real - only the
codec *backend* (`FileFormat`'s dispatch and the four format implementations) is hand-written.

**Translated for real** (swt invocation in `port.sh`): `ImageLoader`, `ImageLoaderListener`,
`ImageLoaderEvent` (extends `java.util.EventObject`, same manual-superclass-embedding pattern as
`TypedEvent`).

**Manual** (`swt/graphics_imagecodec_manual.go`, replacing `graphics_stubs_manual.go`'s
`ImageDataLoaderLoad` stub and `manual.txt`'s `org.eclipse.swt.internal.image.FileFormat`/
`org.eclipse.swt.graphics.ImageDataLoader` entries): `ImageDataLoaderLoad` (the `ImageData(stream)`/
`ImageData(filename)` entry point), `ImageLoader.LoadByZoomStub` (method-level manual override,
`loadByZoom`'s Stream/Optional/`DPIUtil.ElementAtZoom<T>`-based HiDPI dispatch has no translator
rule and is out of scope - the override decodes every frame directly instead),
`NativeImageLoaderSave` (`org.eclipse.swt.internal.NativeImageLoader`, a cocoa PI-layer file never
translated - `ImageLoader.save`'s real entry point), `FileFormatIsDynamicallySizableFormat`/
`FileFormatCanLoadAtZoom` (both real functions kept for the `FileFormat` manual-type name, always
`false` - none of the 4 stdlib-backed formats support arbitrary-size decode, and the zoom one's
call site already fails to build its own `ElementAtZoom` argument and never actually runs it).

### The stdlib codec wrapper

`decodeImages([]byte) []*ImageData` tries `gif.DecodeAll` first (its own multi-frame API - a
non-GIF input just fails to parse and falls through), then `image.Decode` for anything its
registered codecs (`image/png`, `image/jpeg`, blank-imported for their `init()` registration, and
`golang.org/x/image/bmp`, which self-registers as `"bmp"`) recognize. Two converters handle the
result: `palettedToImageData` for a `*image.Paletted` frame (GIF, and a palette PNG/BMP - `image.
Decode` already returns `*image.Paletted` for those) builds an indexed `PaletteData` from `img.
Palette`, always at 8-bit depth (real SWT picks the smallest of 1/2/4/8 that fits the palette
size; `ImageData.SetPixels`/`GetPixels` pack/unpack correctly either way, so this is a size, not a
correctness, difference) and a `transparentPixel` from the one palette entry whose alpha is 0 (a
binary alpha in the palette itself - `image/gif`'s own decode already bakes GIF transparency this
way). `directToImageData` handles everything else as 24-bit direct-color RGB (`PaletteData
(0xFF0000, 0xFF00, 0xFF)`, matching how `PNGFileFormat.java`'s own read code laid out RGB bytes) -
real SWT keeps 8-bit grayscale as an indexed gray-ramp palette instead of promoting it to RGB, one
converter here for every non-indexed source. It buffers each row's pixels/alphas before deciding
whether to keep the alpha plane: a decoder's pixel Go type (`NRGBA` vs `RGBA`) says a pixel *can*
carry alpha, not that the source image actually varies it (a 32-bit BMP decodes as `NRGBA` with
every alpha byte still 255) - only a genuinely non-opaque image gets `SetAlphas` calls and a
non-nil `AlphaData`, checked per pixel. `imageDataToImage` is the inverse, for `save()`: an
indexed `ImageData` becomes `*image.Paletted`, everything else `*image.NRGBA` (`GetPixel`/
`GetAlpha` already unpack 1/2/4-bit indexed and direct-color alike, so one converter covers both).
`gifEncode`'s single-frame case is `gif.Encode` directly (it quantizes any `image.Image` itself);
multi-frame goes through `gif.EncodeAll`, which requires already-`*image.Paletted` frames, so a
frame that came from a direct-color `ImageData` is quantized first with a plain nearest-new-or-
exact-match 256-color builder (`quantizeToPaletted` - no dithering, no popularity analysis; GIF
itself requires an indexed palette per frame, so real SWT's own GIF encoder faces the same
constraint).

### java.io and the resource registry

`internal/jrt/io.go`: `InputStream`/`OutputStream` are Go interfaces mirroring the Java methods
actually used - `Read()/ReadRange(b []int8, off, length int32)/Close()` and `Write(b int32)/
WriteRange(...)/Flush()/Close()` (`[]int8` matches this codebase's existing `byte[]` mapping, not
Go's own `[]byte`). `NewInputStream(io.Reader)`/`NewOutputStream(io.Writer)` wrap a Go stream;
`AsWriter(OutputStream) io.Writer` wraps the other way for handing a `jrt.OutputStream` to a Go
stdlib encoder that wants a real `io.Writer` (`png.Encode`, `jpeg.Encode`, `bmp.Encode`,
`gif.Encode`(`All`) - see "The stdlib codec wrapper" below). `IOException`
is a pointer type (`*jrt.IOException` implements `error`) constructed with no message (every `new
IOException()` in scope is 0-arg). `ReadAllBytes(InputStream) []int8` is `readAllBytes()`.
`NewFileInputStream`/`NewFileOutputStream(filename string)` back `new FileInputStream/
FileOutputStream(filename)` (`ImageLoader.load/save(String)`) via `os.Open`/`os.Create` - the
`"FileInputStream"`/`"FileOutputStream"` Manual names feed `ctorFuncName` only, no such Go type
exists, both constructors return the `InputStream`/`OutputStream` interface directly.

`internal/jrt/resources.go`: `RegisterResources(fs.FS)`/`GetResourceAsStream(name string)
InputStream` - `Class.getResourceAsStream(name)`'s Go stand-in. Simplest version, as directed: one
process-wide `fs.FS` (set once, typically from a hand-written command's own `//go:embed`), `nil`
on a missing FS or a missing name (matching Java's own "resource not found" `null` return). Not a
translator concern - `getResourceAsStream` is only ever called from hand-written Go (`cmd/images`,
the test), never from a file j2go translates.

### Translator changes (general rules, not per-file)

- **`Manual.java`**: `java.io.InputStream`/`OutputStream` registered as ordinary field/param types
  (`isValueType=true`, a bare `jrt.InputStream`/`OutputStream` - same reasoning as `java.lang.
  Throwable` -> `error`), `java.io.IOException` (`*jrt.IOException`), `java.io.FileInputStream`/
  `FileOutputStream` (constructor-name-only aliases, see above), `java.io.BufferedInputStream`
  (aliased to `InputStream` - it only ever appears inside `Image.java`'s own already-dead HiDPI
  `new BufferedInputStream(...)`, whose enclosing `var stream jrt.InputStream = ...` stopped
  compiling the moment `InputStream` became a real type instead of `any` - aliasing the never-
  actually-constructed wrapper type is simpler than teaching the panic-closure machinery to adapt
  an unresolved type's closure to its assignment target's declared type). Two new `MANUAL_METHODS`
  entries (`ImageLoader#loadByZoom`, `NativeImageLoader#save`), erasure-key-style like the existing
  ones.
- **`ExpressionEmitter.fieldGoName`**: an unqualified reference to a field declared on a manual-
  super type (`EventObject.source`) now keeps that type's own Go name (always exported, since
  `jrt` is a separate package - an unexported field there wouldn't even be visible from `swt`'s
  promoted-field access) instead of recomputing one from the Java field's own (here `protected`)
  visibility. Found by `ImageLoaderEvent.toString()`'s `"source=" + source` - the first translated
  file to reference an inherited `EventObject` field unqualified; `TypedEvent` (translated since
  Round 3) never does, only ever calling `GetSource()` or constructing via `jrt.NewEventObject`.
- **`ControlFlowEmitter.emitCatchDispatch`**: a single concrete catch alternative (`catch
  (IOException e)`) now emits a real type assertion (`e := r.(*jrt.IOException)`) instead of a
  bare `e := r` keeping `r`'s static `any` type - the body can then use `e` as that type (e.g.
  pass it where `error` is wanted, as `SWT.error(code, throwable)` does throughout `ImageLoader.
  java`). A true multi-catch of unrelated concrete types still can't have one assertion and keeps
  the old `any` shape. No file before this round had a non-broad (not `RuntimeException`/`Error`/
  `Exception`/`Throwable`) concrete catch at all, so this path was unexercised.
- **`JdkIntrinsics`**: `InputStream.readAllBytes()` -> `jrt.ReadAllBytes(recv)` - `Image.java`'s
  own HiDPI provider path (Round 7 gfx) calls it on an abstractly-typed `InputStream`; the generic
  `Manual.isManual` dispatch would instead emit `recv.ReadAllBytes()`, a method `jrt.InputStream`
  doesn't declare (it only needs `Read`/`ReadRange`/`Close` for the rest of this port).
- **`internal/jrt/util.go`**: `List.Remove(v any) bool` (`java.util.List.remove(Object)`) -
  `ImageLoader.removeImageLoaderListener` is the first translated caller.

### Differences from Java SWT ImageLoader

| Format | Load | Save | ImageData fields that may differ |
|---|---|---|---|
| PNG | yes (`image/png`) | yes (`image/png`) | Always 24-bit RGB + separate `alphaData`, even for an 8-bit palette PNG that `image.Decode` returns as `*image.Paletted` (that case IS indexed, matching SWT); 16-bit-per-channel PNGs decode via Go's own 16-to-8 truncation, not SWT's rounding; interlaced (Adam7) PNGs decode correctly (`image/png` handles it) but never fire `ImageLoaderListener`'s progressive-load callback - decode is all-at-once. |
| GIF | yes (`image/gif`, multi-frame) | yes (single frame: `gif.Encode`; multi-frame: `gif.EncodeAll` after quantizing any non-indexed frame, see above) | `disposalMethod`/`delayTime` copied directly from `gif.GIF.Disposal`/`.Delay` (both already GIF-spec units, no conversion needed); `ImageLoader.repeatCount`/`logicalScreenWidth`/`logicalScreenHeight` copied from `gif.GIF.LoopCount`/`.Config` the same way. Always 8-bit indexed (real SWT can pick 1/2/4/8). |
| BMP | yes (`golang.org/x/image/bmp`: 1/4/8/16/24/32-bit) | yes (`golang.org/x/image/bmp.Encode`, always 32-bit direct color - real SWT's own writer picks a matching depth/RLE compression) | 1/4/8-bit BMPs decode as indexed (matches SWT); 16/24/32-bit as 24-bit direct RGB + `alphaData` only if a 32-bit BMP's 4th byte actually varies (many are a padding byte, always 255 - checked per pixel, not assumed from the bit count). |
| JPEG | yes (`image/jpeg`) | yes (`image/jpeg`, default quality) | Always 24-bit RGB (JPEG has no palette or alpha channel in this port's scope either way, so this matches SWT). `ImageLoader.compression` (SWT's 1-100 JPEG quality knob) isn't read - `jpeg.Encode`'s `nil` options is `jpeg.DefaultQuality` (75, same as SWT's own default). |
| ICO, TIFF, OS/2 BMP, SVG | no | no | `image.Decode` doesn't recognize any of these (no registered Go codec) - `SWT.error(SWT.ERROR_UNSUPPORTED_FORMAT)`, the same error SWT itself raises for a genuinely unrecognized format. |

### Open gaps

`ImageLoader.loadBySize`/`canLoadAtZoom`/`isDynamicallySizable` (the HiDPI `@2x`-variant API,
`DPIUtil.ElementAtZoom<T>`/`java.util.Optional`/`Stream`-based, no translator rule for generic
records or the Stream API) stay unresolved-call panics, same as `Image`'s own `ImageFileNameProvider
`/`ImageDataProvider` HiDPI construction path since Round 7 gfx - none of this is on the `ImageData
(stream)`/`Image(display, imageData)` path the task targets. `java.io.BufferedInputStream`'s alias
is name-only (see above) - a real `mark`/`reset`-based pushback buffer is not ported, matching
`FileFormat.isDynamicallySizableFormat`'s own (already-stubbed) reliance on `mark`/`reset`.

## Round 10 controlexample

**Status: done, green.** `make gen && make` pass (vet, test, every `cmd/*`). `internal/cocoa` is
byte-identical. `bin/controlexample -snap <dir>` opens SWT's ControlExample with 6 tabs (Button,
Canvas, Group, Label, Menu, Text), selects each tab through `NSTabView selectTabViewItemAtIndex:`
(TabFolder's own delegate path), writes `<dir>/<tab>.png` via `cacheDisplayInRect:toBitmapImageRep:`,
and on Button/Canvas/Label/Text clicks one style or option checkbox (`SWT.BORDER`, `Caret`,
`SWT.SEPARATOR`) with `performClick:` - the example recreates its sample widgets (Button: the old
"One" button is disposed, a new one exists) and the snapshot `<dir>/<tab>_<checkbox>.png` shows
the result. Without arguments it stays up like the Java `main`.

### A third Go package: `examples/controlexample`

`port.sh` has a third j2go invocation: ControlExample, Tab, AlignableTab, ScrollableTab and the 6
tabs are the emit files; the whole swt file set (`SWT_FILES`, now an array shared by both
invocations), `C.java` and the cocoa files are reference-only (Round 4's `--`), so every swt name
resolves exactly as when swt was generated. `examples/org.eclipse.swt.examples/src` joined
`Main.SOURCE_ROOTS`. The other 20 tab classes are on the sourcepath (JDT resolves them) but not
translated.

- **Package routing** (`GoTypes.goPackageDir(javaPackage, topLevelName)`): `org.eclipse.swt.
  internal.cocoa` and PI's `org.eclipse.swt.internal.C` -> `internal/cocoa`;
  `org.eclipse.swt.examples.<x>` -> `examples/<x>` (package `<x>`); everything else -> `swt`.
  Before, all of `org.eclipse.swt.internal` went to cocoa, which put
  `org.eclipse.swt.internal.TransparencyColorImageGcDrawer` (common code using swt types) on the
  wrong side. `GoTypes.importPath` maps a package name to its import path.
- **Layering guard** (`PackageQualifier`): cocoa < swt < examples; a file may only qualify a type
  from a lower layer (was: cocoa must not reference swt).
- **Calls into another package's cascade** (`InvocationEmitter`): a dispatch name (`setText_`) and
  the `impl` field are unexported, so a call from another package to a split-dispatch cascade
  method uses the override point's exported wrapper (`button.SetText(...)`), which dispatches
  through `impl` itself.
- **`Impl()` accessor** (`ClassEmitter`): every split-dispatch root with an `impl` field gets
  `func (this *Widget) Impl() WidgetImpl { return this.impl }`, what `.Impl()` in cross-package
  instanceof/cast helpers already expected (cocoa's `id` had a hand-written one). 9 swt roots.
- **SWT constants cross-package** (`EmitUtil.staticFieldGoName`/`staticMethodGoName`): the bare
  `SWT.x` name is qualified too (`swt.PUSH`, `swt.GetPlatform()`).
- **Widened params cross-package** (`EmitUtil.publicParamList`): `swt.CompositeLike` and
  `*swt.Composite`, qualified.
- **Anonymous subclass of another package's class** (`FunctionalEmitter.emitStructAnon`): `init<X>`
  and `impl` are unexported there, so the base is built by the public constructor and copied in
  (`anon.SelectionAdapter = *swt.NewSelectionAdapter()`). A base inside an impl cascade stays an
  `AnonymousClass` marker (none in this file set).
- **`splitsDispatch`** is true for every non-cocoa root, so the example's own Tab hierarchy uses
  the Round 9 api shape (unexported dispatch `createExampleWidgets_`, exported wrapper).

### Translator changes (general rules)

- `ClassEmitter.defaultForwarders`: an abstract class that leaves an interface method to its
  subclasses gets a panicking stub for it, so it satisfies the Go interface its own default
  forwarders pass `this` as (`TransparencyColorImageGcDrawer` leaves `ImageGcDrawer.drawOn`).
- `NumericEmitter.emitInfix`: Java ranks `&`/`|`/`^` below comparisons and `<<` below `+`, Go the
  other way round. A nested infix operand is parenthesized wherever Go would regroup it
  (`shells[i] != null & !shells[i].isDisposed()`). Boolean `&`/`|`/`^` become `&&`/`||`/`!=`
  (short-circuiting - differs only when the right operand has side effects). No existing swt
  output changed.
- `FunctionalEmitter.bodyText`: a void lambda whose body is an assignment or `++`/`--`
  (`e -> untypedEvents = box.getSelection()`) is emitted as that statement.
- `ExpressionEmitter`: `X.class` -> `reflect.TypeFor[*X]()`.
- `ControlFlowEmitter.catchGoType`: a caught translated exception type is package-qualified.
- `InvocationEmitter`: a static call on a manual type records its import.
- `Manual.isBareAny`: `java.lang.Class` (`reflect.Type`) has no Java method surface - calls other
  than the intrinsics are unresolved markers instead of non-compiling `recv.IsArray()`.
- `JdkIntrinsics`: `String.isEmpty`, `String.indexOf(s, from)` (`jrt.IndexFrom`),
  `Integer.parseInt` (`jrt.ParseInt`, panics `*jrt.NumberFormatException`), `Integer.toString(i)`/
  `Boolean.toString(b)`, `Class.getResourceAsStream` (`jrt.ClassGetResourceAsStream`, the class
  is ignored - one resource FS per process), `Throwable.getCause` (`errors.Unwrap`). These also
  resolved 10 old swt markers (`FontData`'s string form and locale parsing, `Display`'s
  line-delimiter conversion): swt markers 130/13/5/2/2 -> 120/13/2/2/2 (MethodInvocation/ClassInstanceCreation/
  CatchClause/ExpressionMethodReference/MethodDeclaration).

### JDK surface (`internal/jrt/text.go`, Manual entries)

`java.util.ResourceBundle` -> `*jrt.ResourceBundle` (`ResourceBundleGetBundle(name)` reads
`<name>.properties` from the registered resource FS; `GetString` panics
`*jrt.MissingResourceException`, which the example's `catch (MissingResourceException e)` catches;
no locale chain), `java.text.MessageFormat.format` -> `jrt.MessageFormatFormat` (`{n}` and `''`
only), `java.lang.NumberFormatException` -> `*jrt.NumberFormatException`. Properties parsing covers
comments, `=`/`:`/space separators, `\` continuation lines, `\t`/`\n`/`\uXXXX` escapes
(`internal/jrt/text_test.go`).

### Newly translated in swt (were manual stubs)

`TransparencyColorImageGcDrawer` (the example's color/font swatches subclass it), `Caret`
(CanvasTab's Caret checkbox creates one; the stub is gone from `widgets_stubs2_manual.go`) and
`ImageUtil` (`createImageRep`, needed by a GC on an `Image(device, ImageGcDrawer, w, h)`; stub gone
from `widgets_stubs3_manual.go`). `tooling/apidump` diff vs the previous round: only additions,
except 7 lines where a stub's signature became the translated one (`Canvas/Decorations/Shell.
SetCaret`, `Display.SetCurrentCaret` take `CaretLike`; `Caret.Release(destroy bool)`,
`Caret.SetFont(FontLike)`, `ImageUtilCreateImageRep(ImageLike, ...)`) - all source-compatible for a
`*Caret`/`*Font`/`*Image` argument. Additions: 88 promoted `Impl()` lines, the `Caret`, `ImageUtil`
and `TransparencyColorImageGcDrawer` APIs.

### Manual (hand-written)

`examples/controlexample/controlexample_manual.go`: `ControlExample.CreateTabs` (replaces
`createTabs()` via `Manual.MANUAL_METHODS`, returns the 6 translated tabs), an opaque `ShellTab`
(`ControlExample.shellTab`'s type; `closeAllShells` is a no-op), the `//go:embed` of the images and
`examples_control.properties` (copied by `port.sh` from the SWT repo), registered from a package
var initializer so it precedes the generated `init()` that calls `ResourceBundle.getBundle`, and
`TabFolder()` for the driver. `cmd/controlexample/main.go` is the Java `main` plus `-snap`.

### Markers (example invocation)

CatchClause 2 (`catch (NullPointerException)` in `getResourceString`), MethodInvocation 22: Java
reflection behind the Set/Get API dialog (`Class.getMethod/isArray/getComponentType`,
`Method.invoke/getReturnType`, `reflect.Array.get/getLength`, `Object.toString` on its results,
`Integer/Long/Character.valueOf` boxing into `Object[]`) and 4 `Object.equals` on widgets in
`handleTextDirection` (only reached when `rtlSupport()`, false on cocoa). None is on the startup or
tab-switch path; opening the Set/Get dialog and pressing Get/Set panics.

### Gaps

- 20 tabs not translated. Widgets they need beyond what swt has: `List` (ListTab), `ProgressBar`,
  `Scale`, `Slider`, `Spinner`, `DateTime`, `Link`, `ToolBar`/`ToolItem` (ToolBar is a stub),
  `CoolBar`/`CoolItem`, `ExpandBar`/`ExpandItem`, `ToolTip`, `Tray`/`TrayItem` (stubs), `Browser`
  (`org.eclipse.swt.browser`), `FileDialog`/`DirectoryDialog`/`PrintDialog` (DialogTab), the
  custom widgets `CCombo`, `CLabel`, `CTabFolder`/`CTabItem`, `StyledText` (the CustomControlExample
  tabs), plus `Tree` columns/`TreeEditor` for TreeTab and `SystemTab`'s `Display` events.
  `ShellTab` itself needs nothing new (Shell styles, `setAlpha`, `Region`).
- The Set/Get API dialog needed reflection (see markers) - implemented, see "Round 10 reflection".
- `Object.equals` on translated objects has no identity rule yet (upcast both sides, compare).
- `MessageFormat` has no format types or quoted sections; `ResourceBundle` no locale fallback.
- Boolean `&`/`|` short-circuit.

## Round 10 reflection

**Status: done, green.** `make gen && make` pass. Fixes the panic from "Round 10 controlexample"'s
own markers: `Class.getMethod`, `Method.invoke`/`getReturnType`, `Class.isArray`/`getComponentType`/
`getName`, `reflect.Array.getLength`/`get`, and a generic `Object.toString()` - the `java.lang.reflect`
subset `Tab.java`'s Set/Get dialog (`getValue`/`setValue`/`getReturnType`/`parameterInfo`) needs on top
of `java.lang.Class` staying `reflect.Type` (unchanged from Round 10 controlexample's `X.class ->
reflect.TypeFor[*X]()`).

**Constraint that shaped the design**: `reflect.Type.MethodByName`/`Method(i)` with a non-constant
name is out - the Go linker sees either call statically and keeps every exported method of every
reflect-reachable type in the binary (dead-code elimination for methods off program-wide), which
would grow every gowt binary a lot more than this fix's own registry does (measured: `cmd/
controlexample` built `-trimpath -ldflags='-s -w'` grew from 8,295,154 to 9,791,298 bytes, +18%,
718 registered methods across 36 widgets-package classes - a `MethodByName`-based version was not
built to compare, but keeping literally every widget/cocoa method reachable from `swt`'s own huge
method surface would be materially larger).

**Design: translator-emitted per-class registry**, not a runtime name-guessing fallback (`ClassEmitter.
registerReflectMethod`/`regParamType`, `internal/jrt/reflect.go`). For every public instance method
of a `org.eclipse.swt.widgets`-package class (own declaration, or a split cascade's override-point
wrapper - the two places a method's real exported Go name is actually emitted), `ClassEmitter` also
emits a line into that class's own `func init()`:

```go
jrt.RegisterMethod(reflect.TypeFor[*Text](), "getSelection", nil, reflect.TypeFor[*Point](),
	func(target any, args []any) any { return jrt.Narrow[*Text](target).GetSelection() })
jrt.RegisterMethod(reflect.TypeFor[*Text](), "setSelection", []reflect.Type{reflect.TypeFor[PointLike]()}, nil,
	func(target any, args []any) any { jrt.Narrow[*Text](target).SetSelectionSelection(jrt.ArgAs[PointLike](args[0])); return nil })
```

The Java method name is the registry key (captured verbatim where the method is emitted, not
reconstructed from the Go name), so naming can never diverge from the real translation rule -
this *is* the "preferred" design from the brief, just scoped to `org.eclipse.swt.widgets` rather
than every translated class (cheap to widen later: same two call sites, same gate). paramTypes/
returnType are the method's own Go types (widened to its `<Class>Like` interface the same way
`publicParamList` does, so `ClassGetMethod`'s overload matching sees the same type an external
caller would pass - `regParamType` mirrors `EmitUtil.publicParamList`'s per-parameter branch).

**`getExampleWidgets()` returns `Widget[]`, i.e. `[]*Widget`** - each element is an upcast
promoted-field address (`&someText.Widget`, see `upcastswtTextToswtWidget`), so `reflect.TypeOf` on
it always reports `*Widget`, never `*Text`: Go has no runtime-polymorphic object identity the way
Java references do. `Object.getClass()`'s own intrinsic (`JdkIntrinsics`) now routes through the
receiver's `.Impl()` (the impl-cascade field every split-dispatch root already has, Round 6+) when
the receiver's static type has one, recovering the real concrete type `reflect.TypeOf(x.Impl())`
- this also fixes `Widget.GetName()`/every translated `toString()`'s own `getClass().getName()`,
previously reporting "Widget" for a `*Text` too, for the same underlying reason.

**`internal/jrt/reflect.go`**: `Method` (a `paramTypes`/`returnType`/closure triple from one
`RegisterMethod` call), `ClassGetMethod(t, name, paramTypes)` (looks `name` up in `t`'s own
registered methods, then its ancestors' - `parentOf` walks the field-0 embedding chain every
translated class embeds its superclass through, struct-field reflection only, never Method/
MethodByName), `Method.Invoke` (panics `*InvocationTargetException` on failure, matching the
"exceptions are panics" contract `jrt.ParseInt`/`ResourceBundle` already use), `Narrow[T]` (the
address of a registered closure's own declaring type within whatever concrete leaf `target`
actually is - what Go's own method promotion does, done explicitly with `Field(0).Addr()` so a
closure registered on `*Control` still works when invoked with a `*Button`), `ArgAs[T]` (one
`Method.invoke` argument, boxed `any`, nil for Java `null`, converted to the closure's own param
type), `ClassName` (`Class#getName()`: JVM primitive names and array-descriptor syntax exactly -
what the dialog's `setValue` branches on - plus a best-effort `org.eclipse.swt.widgets.<Name>`
guess for any other pointer-to-struct type, harmless since every existing `getClass().getName()`
caller only keeps the tail after the last `.`).

**`JdkIntrinsics` additions**: `Integer.valueOf`/new `Long.valueOf`/`Character.valueOf` (the
dialog's `setValue` boxes a parsed numeric/char into `Object[]` - previously only the
primitive-argument overload of `Integer.valueOf` was handled), `java.lang.reflect.Array.getLength`/
`get` (plain `reflect.ValueOf(...).Len()`/`Index(...).Interface()` - Java arrays are Go slices
throughout this port already, nothing array-specific to add).

**Verify**: `internal/jrt/reflect_test.go` (a hand-written two-level struct hierarchy, no
`Display` needed - `ClassGetMethod`/`Invoke`/`GetReturnType` on an own method, an inherited one via
the field-0 climb, and two overloads of the same Java name disambiguated by paramTypes; `ClassName`
on primitives/arrays). Live: `ControlExample`'s Text tab, calling `Tab.GetReturnType`/
`ParameterInfo` directly (no `Display` panic) returns correct Java-shaped descriptions for
`Text`/`Selection`/`ToolTipText`/`TextChars` (`String`/`Point`/`String`/`char[]`); clicking the real
"Set/Get API" button opens the dialog and populates its combo/labels/set-button/get-button text
via the real reflection chain for the default property (`DoubleClickEnabled` -> `boolean e.g.
true`).

**Found, not fixed - blocks full dialog exercise**: `Tab.resetLabels()`/`getValue()` call
`setText.SetText("")`/`getText.SetText("")` as their own first statements (matching Java's
`setText.setText("");`) - `Text.SetText` panics `*jrt.IllegalArgumentException` (`ERROR_NULL_
ARGUMENT`) for an empty string. Root cause: `NumericEmitter`'s `x == null` -> `x == ""` rule for
Go `string` operands (`string` has no nil to compare against) - a deliberate, already-relied-on
simplification elsewhere, but it makes `Text.setText("")` (a legal, common "clear the field" call
in real SWT) indistinguishable from `setText(null)`. Pre-existing (present since `Text.java` was
first translated), unrelated to this round, newly exposed because this round is the first thing to
make `resetLabels()` run past its own `parameterInfo()` call. Not fixed here: the type model has no
way to represent Java's null distinctly from `""` for a Go `string` without a broader change (e.g.
`*string`) touching every String-typed field/param across the whole translated set. Blocks
`GetValue()`/`SetValue()`/`ResetLabels()` specifically (not `GetReturnType`/`ParameterInfo`, which
this round's live verification used instead) for every tab, every property.
Fixed in "Round 11 null-string and reflect opt-in".

## Round 11 null-string and reflect opt-in

**Status: done, green.** `make gen && make` pass, `internal/cocoa` byte-identical, every `cmd/*` runs
3 s without a crash.

### Reflect registry is opt-in (`ReflectEmitter`, `swt/swtreflect`)

Round 10 put the 718 `jrt.RegisterMethod(...)` calls into each widgets class's own `init()` in
package `swt`, so every program linking `swt` kept all those methods. Now `ReflectEmitter` (split
out of `ClassEmitter`; `ClassEmitter`/`Emitter` call `emitter.registerReflectMethod` at the same two
points as before) collects them across the whole run, and `Main` writes one generated file,
`swt/swtreflect/swtreflect.go` (package `swtreflect`, one `func init()`), only when the run
registered something - the cocoa and example runs leave it alone. The registration text is built
with `currentGoPackage = "swtreflect"` and its own import set, so the ordinary qualification rules
spell every type as seen from outside `swt` (`*swt.Text`, `swt.PointLike`, `jrt.Runnable`). All 718
entries call exported API only (the natural-name method or the override-point wrapper), none needed
rerouting.

A program that needs Java-style reflection imports it for the side effect:
`import _ "github.com/haiodo/gowt/swt/swtreflect"` - `examples/controlexample/controlexample_manual.go`
does (the Set/Get API dialog).

- `PackageQualifier.qualifyManual` (from `GoTypes.map`): a hand-written type spelled without a package
  (`Accessible`, `IME`, `Tray`, `ToolBar`, `TaskBar`, `AutoscalingMode`, an example's `ShellTab`)
  lives in its Java package's Go package and is qualified when referenced from a higher layer.
  Spellings with a dot (`jrt.X`) or lowercase (`any`, `error`) are left alone. No existing output
  changed; the registry uses it for 9 entries.
- `JdkIntrinsics`: `Method.invoke(target, ...)` passes `target.Impl()` when the target's static type
  has an impl cascade - the same rule `getClass()` already used. `getExampleWidgets()` returns upcast
  `*Widget` addresses, and `jrt.Narrow[*Text]` can only walk from the concrete object up to its
  ancestors, so `Invoke(widgets[i])` failed with `*int32 does not embed *swt.Text` (not reached in
  Round 10, `getValue()` panicked before).

Release sizes (`CGO_ENABLED=0 go build -trimpath -ldflags='-s -w'`):

| binary | Round 10 | Round 11 |
|---|---|---|
| `cmd/hello` | 8,930,754 | 6,386,946 |
| `cmd/controlexample` | 9,791,298 | 9,561,394 |

`hello` is back to its pre-reflection size (6,386,834 + 112). `controlexample` still carries the
registry because it imports `swtreflect`.

### Java null vs "" for String (`NumericEmitter.stringParamNullCheck`)

Go's `string` has no nil; the port keeps `""` as null's stand-in (`adaptNumeric` turns a `null`
argument/assignment into `""`, `x == null` into `x == ""`). That made SWT's argument validation fire
for the empty string: `Text.setText("")` panicked with `ERROR_NULL_ARGUMENT`.

Rule as implemented: a `==`/`!=` comparison of a **String parameter** with `null` becomes the constant
`false`/`true` when it is a **null-argument guard** - the condition of an `if` (alone, parenthesized,
or one operand of a `||` chain) whose then-branch is `error(...ERROR_NULL_ARGUMENT)` (either
`error`/`SWT.error`) or a `throw`. A Go caller cannot pass null, so such a guard can never be true.
Every other String null comparison keeps `== ""`.

Why not every parameter null check: several parameters use null as "absent" and internal callers
pass it (translated to `""`). With a blanket rule these changed behaviour:
`Device.getFontList(faceName)` (null = all fonts, would return none), `Font.init(..., nsName)` (null
= no native name, would call `fontWithName:` with ""), `MenuItem.setToolTipText(null)` (clears the
tooltip). `FontData.setLocale`, `SWT.error(code, t, detail)` were harmless but are unchanged too.

Affected sites (39, all `if false { Error/this.Error(ERROR_NULL_ARGUMENT) }` now):
- widget text setters: `Button/Label/Group/Combo/Text.SetText`, `Item/Decorations/Shell/MenuItem/
  TabItem/TableColumn/TreeColumn.setText_`, `TableItem/TreeItem.SetTextIndexString`, `Dialog.SetText`,
  `MessageBox.SetMessage`, `Text.SetMessage/Append/Insert`;
- `Combo` item API: `Add`, `AddStringIndex`, `IndexOfStringStart`, `RemoveString`, `SetItem`;
- data keys: `Widget.GetDataKey/SetDataKeyValue`, `Display.GetData/SetData`;
- graphics: `Font.Init(name)`, `FontData(String)`/`SetName`, `GC.DrawTextStringXYFlags`/
  `TextExtentStringFlags`, `TextLayout.SetText`, `Device.LoadFont`, `Image(device, filename)`,
  `ImageLoader.LoadByZoom.../SaveFilenameFormat/CanLoadAtZoom...`.

Kept as `== ""` (null and "" merge): String **fields** (e.g. `this.text != ""` before drawing, `toolTipText == ""` in Table/Tree's expansion frame,
`displayText`, `FontData.lang/country/variant`), **locals** and **returns** (`Text.setText`'s
`string = verifyText(...); if (string == null) return` - a verify listener returning "" now also
returns early, same as before), and null-as-absent parameters above. None of these needs to tell
null from "" on a path a Go caller can reach, so no explicit null marker was introduced.
Short-circuits stay correct: a guard operand becomes a literal inside the same `||`
(`if false || x == null`), and `s == null || s.length() == 0` outside a guard is unchanged.

Verify: `swt/graphics_test.go` `TestFontDataSetNameEmpty` (`FontData.SetName("")`, no Display;
panicked before). Live: `bin/controlexample`, Text tab, "Set/Get API" dialog, property `Text`: Get
shows the example Text's content, Set "Hello gowt" changes the widget and Get shows it, Set "" clears
the widget and Get shows "" - no panic.

## Round 12 tests

**Status: done, green.** `make gen && make` pass, `make test-swt` runs to completion: 352 tests, 292
passed, 55 failed, 5 skipped (`tests/RESULTS.md`). `internal/cocoa` byte-identical, every `cmd/*` runs.

### Layout

- `port.sh` has a fourth j2go invocation: the test files (`TEST_FILES`: `SwtTestUtil`, `ImageTestUtil`,
  `CapturedOutput`, the `graphics`/`layout`/`events` test classes, `tests/graphics/ImageDataTestHelper`)
  are emitted; swt, `C.java` and cocoa are reference-only. `Test_org_eclipse_swt_layout_BorderLayout` is
  left out (see `tests/RESULTS.md`). The test images are copied to `tests/swttests/testdata/` and
  embedded by `tests/swttests/swttests_manual.go` (`Class.getResourceAsStream`).
- `org.eclipse.swt.tests.*` -> `tests/swttests` (package `swttests`, `GoTypes.goPackageDir`), layer 2
  like the examples.
- JUnit on the parser classpath only: `Main --classpath <jars>`; `pom.xml` declares
  `junit-jupiter-params` 5.11.4 as `provided` so Maven fetches it (with api, opentest4j,
  platform-commons, apiguardian) without shading it; `port.sh` builds the path from `~/.m2`.
  `org.eclipse.test.Screenshots` (not in the SWT repo) is a parser-only stub under
  `tooling/j2go/stubs` (an extra source root).
- In `swttests` a type that is neither translated nor manual degrades to `any`, like a JDK type
  (`Emitter.degradesUnresolvedTypes`): tests may reference SWT API not ported yet; such a test
  compiles and fails at run time with an `unresolved` marker. Other packages keep the
  `unsupported_type_` guard.

### `internal/junit` and the registry (`TestEmitter`)

- `Assertions.x(...)`/`Assumptions.x(...)` -> `junit.X(...)` (`Emitter.tryIntrinsic` asks
  `TestEmitter.junitCall` first). The shim takes `any` plus JUnit's trailing message (`msg ...any`: a
  string or a `func() string` Supplier). A numeric argument is converted to its Java type
  (`int32(100)`) - an untyped Go constant would arrive as `int` and never equal an `int32`.
  Overloads with a `double`/`float` third parameter map to `AssertEqualsDelta`/`AssertNotEqualsDelta`/
  `AssertArrayEqualsDelta`. `assertThrows(X.class, exec)` and `assertInstanceOf(X.class, v)` take the
  class as a Go type argument: `junit.AssertThrows[*swt.SWTException](func() {...})`, returning the
  exception. A lambda or bound method reference argument is a bare Go func (`FunctionalEmitter.rawFunc`),
  not a functional-interface adapter.
- Semantics: `AssertEquals`/`AssertArrayEquals` use the translated `Equals(any) bool` when the
  expected value has one, identity otherwise; identity (`AssertSame`) compares pointer addresses, since
  an upcast is the address of the embedded superclass field (offset 0) and has another Go type. Null is
  a nil pointer/slice/map/func/interface or `""` (the port's null String). A failed assertion panics
  `*junit.AssertionFailed`, a failed assumption `*junit.Skipped`.
- Per concrete top-level test class (`ClassEmitter` calls `TestEmitter.registration` after the class):
  a `func init() { junit.Register(&junit.Class{...}) }` with `New`, `BeforeAll`/`AfterAll` (static),
  `BeforeEach` (superclass first)/`AfterEach` (subclass first) and `Tests`. Methods are each signature's
  most-derived declaration along the superclass chain, so inherited tests run on the subclass and an
  override without `@Test` is not a test (JUnit 5 rules). A call goes through the impl cascade like any
  call (`Emitter.instanceCall`), so an overridden `newTypedEvent` dispatches to the subclass.
- Annotations, evaluated at translation time for macOS: `@Test`; `@ParameterizedTest` +
  `@ValueSource` expanded into one test per value (`name[1]`, ...); `@Tag`/`@Tags` (class tags inherited);
  `@Disabled`, `@DisabledOnOs`/`@EnabledOnOs` (`OS.MAC`), `@DisabledIfEnvironmentVariable` (checked at
  init by `junit.SkipIfEnv`), `@DisabledIfSystemProperty` (never: Go has no system properties),
  `@Timeout` (per-test watchdog), `@TestMethodOrder` with `OrderAnnotation`/`MethodName`. A test with
  another `org.junit` annotation, another parameter source (`@MethodSource`) or injected parameters
  (`@TempDir Path`, `TestInfo`) is registered as skipped with the reason.

### Runner `cmd/swttest`

`runtime.LockOSThread()` in `init`; one `Display` created up front on the main thread (recreated if a
test disposed it). Each test gets a fresh instance; `BeforeEach`, the test and `AfterEach` each run under
`recover` - a panic fails that test, `AfterEach` still runs. A `time.AfterFunc` watchdog per test
(`-timeout`, default 30s, or the test's `@Timeout`) prints the hung test as FAIL and exits 2. Flags:
`-run <regexp on Class.method>`, `-tag a,b,!c`, `-json` (`go test -json` events: run/output/pass/fail/skip
per test, a package pass/fail at the end), `-list`. One line per test: `PASS|FAIL|SKIP Class.method
(0.012s): message at file:line` - the site is the first frame outside the runtime, the shim and the
runner. Exit 0 when the run completes, failures included (TSK-053 adds the expected-results gate).
`make test-swt` builds and runs it (`SWTTEST_FLAGS=...`).

### Translator changes (general rules)

- `ControlFlowEmitter.emitTry`: `finally` (and try-with-resources closes) ran as function-level
  defers, i.e. at method exit - late, and a later reassignment of the same variable was what the
  deferred code saw (`Font.getFontData` test: five sequential try/finally disposed the last font five
  times). Now only the method's last statement keeps the inline form (a function-level defer then
  runs right after it); any other try runs in a closure whose defers fire when the try ends. The body
  is always block-scoped (Java allows the same local name in sibling try blocks). Resource closes are
  separate defers, the last declared closing first, so a close that panics still lets `finally` run
  (`EventTable.sendEvent`'s `ExceptionStash`, `eventtable_test.go`). Also stops
  `ControlExample.initResources` from keeping every image stream open until the method returns.
- `StatementEmitter`: a local that Java only ever assigns gets `_ = x` (`EmitUtil.neverRead`); a
  for-each over a `java.util` collection (`*jrt.List`) ranges over `ToArray()` with a `jrt.Cast` to the
  loop variable's type; the unsupported-for-each marker is `func() { panic(...) }()`, not a terminating
  statement, so `go vet` does not report the code after it as unreachable.
- `FunctionalEmitter`: `Type::method` (JDT parses it as an `ExpressionMethodReference` whose expression
  is a type name) is the Go method expression `(*swt.Image).Dispose` or the static function, not
  `Image.Dispose`; a void lambda with a non-statement expression body (`() -> rect.x`) is `_ = expr`
  instead of a marker.
- `ClassEmitter`: a nested interface is emitted as a Go interface (was a struct).
- `ExpressionEmitter`: a static field of an unresolved JDK class with a compile-time constant value
  (`Short.MAX_VALUE`) is that literal; any other unresolved static field is a typed panic closure
  instead of an undefined identifier; an anonymous class marker has its base type (was `any`, did not
  compile where a `*swt.Canvas` was expected). A manual type's static member is package-qualified from
  another package (`swt.DPIUtilGetDeviceZoom()`; `InvocationEmitter` too).
- `InvocationEmitter`: `new X(args)` of a value-type manual class (`Thread` = `any`) still evaluates
  its arguments.
- `Names.javaMethodBaseGoName`: `$` (legal in Java identifiers, used in JUnit method names) -> `_`.
- `JdkIntrinsics`: `String.startsWith/endsWith/contains/toLowerCase/toUpperCase` (`strings.X`),
  `String.hashCode` (`jrt.StringHashCode`), `Double.hashCode`, `Objects.hash`, `System.lineSeparator()`,
  `System.getProperty("os.name")` = `"Mac OS X"`. swt markers: MethodInvocation 120 -> 115,
  ClassInstanceCreation 13 -> 9.
- `Manual`: `java.io.ByteArrayInputStream`/`ByteArrayOutputStream` -> `jrt`.

### `internal/jrt`

`List.ToArray/Contains/ForEach`, `ListOf`, `Map.ForEach` (the action arrives as an erased func and is
called through reflect), `ByteArrayInputStream`/`ByteArrayOutputStream` (`ToByteArray`, `ToString`),
`StringHashCode`/`DoubleHashCode`/`ObjectsHash`. `NewIllegalArgumentException` returns a pointer (it
returned a struct, so `panic(jrt.NewIllegalArgumentException(..))` never matched a
`catch (IllegalArgumentException)`, whose type is `*jrt.IllegalArgumentException`) and takes Java's four
constructor shapes (`()`, `(String)`, `(Throwable)`, `(String, Throwable)`).

### Gaps

- Widget test classes: not translated (task 052). A trial run over all of them stops at JDT errors
  (`CoolBar`, `CoolItem`, ... are not on the cocoa source path).
- `@MethodSource`/`@CsvSource`, parameter injection (`@TempDir`, `TestInfo`), `@RegisterExtension`:
  skipped with a reason.
- `object == null` on `any` holding a typed nil pointer is false (`tests/RESULTS.md`, translator bug),
  not fixed here: it is `NumericEmitter.emitInfix`, which another branch is changing.

## Round 12 semantics

**Status: done, green.** `make gen && make` pass, `internal/cocoa` byte-identical, every `cmd/*`
runs 3 s without a crash, `bin/controlexample -snap` unchanged. Two general Java-semantics
deviations, both flagged as open gaps in "Round 10 controlexample".

### Boolean `&`/`|`/`^` and `&=`/`|=`/`^=` no longer short-circuit

Java's boolean `&`/`|` (and their compound-assignment forms) always evaluate both operands; Go's
`&&`/`||` stop at the first false/true. `NumericEmitter.goOperator` already mapped `&`/`|`/`^` to
`&&`/`||`/`!=` (Round 10) - `!=` is fine (not short-circuiting in Go either), but `&&`/`||` drop a
right-hand side effect whenever the left operand alone already decides the result.

`NumericEmitter.hoistBooleanChainIfNeeded` (called from `emitInfix` right after `goOperator`):
for a boolean `&`/`|` chain (2 or more operands, `extendedOperands()` included) where any operand
past the first can have a side effect (`NumericEmitter.hasSideEffect`: a call, `new`, an
assignment, or `++`/`--` - reuses the same shape as `ExpressionEmitter.containsCall`, extended),
every operand is hoisted into its own `bN := <operand>` prelude line (existing prelude mechanism,
README "instanceof / pattern matching"), then combined with `&&`/`||` over the temps. A
side-effect-free chain is untouched (still plain `&&`/`||`).

```go
// before (Round 10): events = events || this.RunTimers() - short-circuits once events is true
// after:
b266 := this.RunTimers()
events = events || b266
```

Same rule for `&=`/`|=` (`StatementEmitter.booleanCompoundOp`, also used by
`ExpressionEmitter.emitInlineAssign`'s inline compound-assign path, which previously emitted the
literal Java operator token - invalid Go for `bool &=`/`bool |=`, never hit by the corpus): a
side-effecting RHS is hoisted into a temp before the `lhs = lhs && tmp`/`lhs = lhs || tmp` line.
`^=` keeps its existing `!=` form (never short-circuits).

**Real site found**: `Display.ReadAndDispatch`'s `events |= this.RunSettings()/.RunTimers()/
.RunContexts()/.RunPopups()/.RunPaint()/.RunDeferredEvents()` chain - once any earlier call in the
chain returned true, every later `Run*()` call was skipped by Go's `||`, silently starving
timers/contexts/popups/deferred-event dispatch for the rest of that `readAndDispatch()`. Fixed by
`swt/widgets_display.go`'s regenerated `ReadAndDispatch`. Also
`examples/controlexample/controlexample_menutab.go`'s `MenuTab.CloseAllShells` (`shells[i] != null
& !shells[i].isDisposed()`, the exact expression named in "Round 10 controlexample").

Test: `swt/identity_test.go` `TestBooleanOrEvaluatesEveryOperand`.

### `Object.equals`/`==` identity through `.Impl()`

Java's default `Object.equals`/`==` is reference identity. With the impl/embedding model (README
"Inheritance / impl dispatch") the same object can be referenced through different embedded-
pointer levels (`&shell.Widget` vs `shell`), and two unrelated sibling classes (`Button`/`Label`)
have no upcast helper connecting them at all - `NumericEmitter.upcastObject`'s ancestor-only
upcast can't reconcile that case, and the receiver fell into InvocationEmitter's generic
"unresolved call" panic marker for `.equals()` (no case existed for it).

**`.equals()`** (`JdkIntrinsics.emitObjectEquals`, hooked in `tryIntrinsic` next to `getClass()`):
fires when the resolved method is `java.lang.Object#equals` (no override anywhere in the
hierarchy). Each side routes through `.Impl()` when its own static type is a translated class with
an impl cascade (`hasImpl`, the same check `getClass()`/`Method.invoke` already use); `.Impl()`
always normalizes to the concrete leaf pointer regardless of hierarchy shape. Both sides are
`any`-boxed (`any(l) == any(r)`) so two different roots' differently-typed `Impl()` interfaces
still compile against each other - no runtime helper needed, `any` comparison already does dynamic
type+value identity. Neither side having impl (e.g. two `String`/struct/manual values wandering
through a raw `Object.equals`) falls back to a plain `recv == arg`.

```go
// examples/controlexample/controlexample_tab.go, Tab.HandleTextDirection - was an unconditional
// panic ("unresolved call equals"), 4 sites:
if any(this.ltrDirectionButton.Impl()) == any(widget.Impl()) {
```

**`==`/`!=`** (`NumericEmitter`, in the existing `EQUALS`/`NOT_EQUALS` block): `upcastObject` is
tried both ways first, unchanged (ancestor/descendant `==` was already correct - Go's promoted-
field addressing gives the same address for the same object down any embedding path, so no output
changed there). Only when upcasting resolved neither side (sibling types, upcast a no-op both
ways), neither operand is a null literal, and the two Go types still differ, does
`identityCompare` kick in with the same `.Impl()`+`any` scheme as `.equals()` above - skipped
whenever either side's Go type is already plain `any` (that already compiles and compares
correctly as-is; this exclusion is what keeps every existing `if (object == this) return true`
fast path in the hand-translated `*.Equals` methods - `Color`, `Font`, `Image`, ... - byte-for-byte
unchanged, since one side there is always the `any`-typed `equals(Object)` parameter). No live
sibling-vs-sibling `==` site exists in the current corpus; this closes a real compile gap (`*Button
== *Label` has no direct upcast relation either way and wouldn't have type-checked) rather than
changing any existing behavior.

Test: `swt/identity_test.go` `TestObjectIdentityThroughImpl`.

### Marker counts (port.sh runs 3 j2go invocations - cocoa, swt, examples - each prints its own summary)

| invocation | kind | before | after |
|---|---|---|---|
| cocoa | `StaticFieldMethodNameClash` | 166 | 166 (unrelated, untouched) |
| swt | `MethodInvocation` (total) | 120 | 116 |
| swt | `MethodInvocation` (`unresolved declaring type java.lang.Object.equals`) | 4 | 0 |
| swt | `CatchClause`/`ClassInstanceCreation`/`ExpressionMethodReference`/`MethodDeclaration` | 2/13/2/2 | 2/13/2/2 (unrelated, untouched) |
| examples | `MethodInvocation` (`unresolved declaring type java.lang.Object.equals`) | 4 | 0 |
| examples | `CatchClause` | 2 | 2 (unrelated, untouched) |

8 `java.lang.Object.equals` markers total -> 0 (confirmed by `grep -r "unresolved call equals" swt
examples`, 8 -> 0). The remaining `CatchClause` markers are unrelated (`catch (NullPointerException)`
in `getResourceString`, README "Round 10 controlexample").

### Changed generated sites (full list)

`.equals()` -> `.Impl()` identity (4, all `examples/controlexample/controlexample_tab.go`
`Tab.HandleTextDirection`) and 4 more `Image.Equals`/`TextStyle.Equals` sites in
`swt/graphics_image.go` (3: `imageDataProvider`/`imageFileNameProvider`/`imageGcDrawer`) and
`swt/graphics_textstyle.go` (1: `Data`), where the compared field is declared `Object`-erased but
resolves to a manual (non-impl) type - these were also `.equals()`-on-`Object` unresolved-call
markers before, now a plain `recv == arg` (see `emitObjectEquals`'s "neither has impl" branch; both
sides are always Go-comparable pointers in this file set, confirmed by `go build`/`go vet`/
`go test` all green).

Boolean-`|=` hoisting: `swt/widgets_display.go` (`Display.ReadAndDispatch`, 6 calls) and
`examples/controlexample/controlexample_menutab.go` (`MenuTab.CloseAllShells`, 1 site).

Every other file in the 30-file diff (`swt/custom_scrolledcomposite.go`, `custom_tableeditor.go`,
`graphics_cursor.go`, `graphics_fontmetrics.go`, `graphics_gc.go`, `graphics_glyphmetrics.go`,
`graphics_imagedata.go`, `graphics_imageloader.go`, `graphics_lineattributes.go`,
`graphics_palettedata.go`, `graphics_path.go`, `graphics_pattern.go`, `graphics_region.go`,
`graphics_textlayout.go`, `widgets_combo.go`, `widgets_dialog.go`, `widgets_fontdialog.go`,
`widgets_messagebox.go`, `widgets_tabfolder.go`, `widgets_table.go`, `widgets_tablecolumn.go`,
`widgets_tableitem.go`, `widgets_tree.go`, `widgets_treecolumn.go`, `widgets_treeitem.go`) only
renumbers `condN`/`okN`/`tN`/`anonN`/`innerN` temp names - collateral from `tempCounter` being a
single counter shared across the whole invocation, shifted by the new hoists earlier in the run;
same file set, same logic, verified via a digit-stripped diff against the pre-Round-12 output.

