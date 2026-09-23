# j2go

## Status / next steps (round 5)

**Parts 1-3: done, verified, green.** `CGO_ENABLED=0 go build ./... && go vet ./... && go test
./...` all pass. `internal/cocoa` unchanged (still all 210 PI/cocoa files + `C.java` in one
invocation). `swt`'s own invocation grew to include `Widget.java`/`Control.java`/`Scrollable.java`
(cocoa) alongside stage-1/round-3's files, with all 210 cocoa files + `C.java` fed after `--` as
reference-only (parsed and modeled for name resolution, not re-emitted - see "Cross-package
qualification" below). 30 Go tests pass, including a new `swt/widgets_widget_test.go`
(`TestWidgetCheckBits`, exercises `WidgetCheckBits`'s mutually-exclusive style-bit resolution -
no `Display` needed).

Round 4 translated `Widget.java` (2452 lines) + `Control.java` (5278) + `Scrollable.java` (439)
for real - the whole widget-core file set the task brief named, not a partial attempt. Getting
there needed real cross-package qualification (direction 4 from Round 2, previously deferred as
YAGNI) plus ~15 further translator fixes (below) and ~350 lines of hand-written manual stubs for
the classes Part 3 (below) still owns: `Composite`/`Canvas`/`Decorations`/`Shell`/`Menu`/
`ScrollBar` (embed their real translated superclass, so promoted fields/methods and the
`upcastObject` mechanism work without restating them), `Accessible`/`ACC`, `WidgetSpy`,
`Callback` (reflection-based dispatch - unsupported marker, not this round's job), `Font`/
`Color`/`Image`/`Cursor`/`Region`/`GCData`/`Dialog`/`Device`/`Resource`/`AutoscalingMode`
(`swt/widgets_manual_stubs2.go`), plus real fields/methods added to the existing `Display`/`GC`
opaque stubs (`swt/widgets_stubs_manual.go`) and `Monitor` (`swt/widgets_monitor_manual.go`).
`Widget` itself is no longer manual - removed from `manual.txt`/`Manual.ENTRIES`.

**Not attempted** (same reasons as Round 3, still true): `synchronized`, `java.util` collections,
`Synchronizer.java`/`RunnableLock.java`. Widget's one generic method
(`getTypedListeners<L>`, `Stream`/method-refs) is skipped entirely, like an abstract method (see
"Generic methods" below) - no Go equivalent for its own signature, not just its body. Control's
one `new Callback(...)` (`getPath`/`regionToRects`) is a manual-stub panic, not a real
reflection-dispatch implementation (see "Callback" in manual.txt above) - matches the task
brief's "do not build the full lambda/callback subsystem this round".

**Part 3 (the rest of widget core): not started.** `Composite.java` 1324 lines, `Canvas.java`
699, `Decorations.java` 718, `Shell.java` 2580 (~5,300 lines) - each currently a manual stub
embedding its real translated superclass; translating them for real means replacing the stub's
panic-bodied methods with the genuine ones and should be a smaller lift than Round 4 (the
cross-package/upcast machinery now exists and is exercised). Also still open: `Layout.java`/
`layout/*` (needs `Composite`/`Control` first), `Item.java` (`Widget` subclass, needed before
`MenuItem` if a later round wants it), `Display.java` (6861 lines, its own manual stub already
carries the ~35 fields/methods Round 4 needed - translating it for real is a much bigger, separate
effort), graphics `Device`/`GC`/`Color`/`Font`/`Image` (each still a thin manual stub).

**Next steps, in order**: (1) `Composite`/`Canvas`/`Decorations`/`Shell` for real, in that
order (each only adds a thin layer over the previous), replacing their manual-stub entries in
`manual.txt`/`Manual.java` one at a time the same way `Widget` was retired this round; (2)
`Layout.java`/`layout/*` once `Composite`/`Control` are real; (3) `Item.java`; (4) only then
`Display.java` and the graphics classes, likely their own multi-round effort; (5) Snippet1.

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
- **new Java construct in a class body** (field, method shell, constructor, super/this call) ->
  `ClassEmitter`
- **new native-method shape** (purego binding, `_sizeof`, `_stret`, constant accessor) ->
  `NativeEmitter`
- **switch/throw/try-catch-finally** -> `ControlFlowEmitter`
- **new expression form** (literal, operator, name/field resolution, string escaping) ->
  `ExpressionEmitter`
- **method invocation / `new` / argument adaptation** -> `InvocationEmitter`
- **instanceof, cast, or the impl-cascade helpers** -> `TypeTestEmitter`
- **try/catch escape detection** -> `EscapeScanner` (the ASTVisitor pre-scan)
- **a stateless text/binding helper shared by several components** -> `EmitUtil`

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

`--out` is the repo root, not a single package directory: each file lands under `swt/` or
`internal/cocoa/` per its own Java package (`Main.goPackageDir`/`goPackageName` - currently a
2-entry mapping, `org.eclipse.swt.internal.cocoa` -> `internal/cocoa`, everything else -> `swt`;
see "Multiple Go packages" below).

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

`org.eclipse.swt.internal.Callback` wraps `(object, methodName, argCount)` and uses JNI-generated
reflection so a Java method can serve as a native function pointer (registered as an ObjC IMP via
`class_addMethod`, ~298 call sites in the cocoa PI layer, but only ~25 actual `new Callback(...)`
sites - one Callback's IMP is multiplexed across many selectors/classes, dispatch happens inside
the callback based on which `self`/`_cmd` it was invoked with). Go has first-class functions, so
none of the reflection machinery is needed: `cocoa.NewCallback(fn any) uintptr` is a 1-line
wrapper over `purego.NewCallback` - an ordinary Go closure with the right signature (self/_cmd as
`uintptr` first, matching class_addMethod's type-encoding string) *is* the trampoline. The
multiplexed-dispatch logic itself (which selector/self maps to which SWT method) is
`Display`/`Widget` bookkeeping, not part of the Callback mechanism - out of scope until Step 3
translates `Display.java`.

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
