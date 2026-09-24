package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import org.eclipse.jdt.core.dom.Expression;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.ITypeBinding;
import org.eclipse.jdt.core.dom.MethodInvocation;
import org.eclipse.jdt.core.dom.QualifiedName;
import org.eclipse.jdt.core.dom.StringLiteral;

/** Mapping of plain JDK calls used by the translated set: Math.min/max, System.arraycopy,
 * System.getProperty("os.arch"), String/Consumer/Throwable methods. The single place to add
 * a new JDK method mapping. */
final class JdkIntrinsics {

	private final Emitter emitter;

	JdkIntrinsics(Emitter emitter) {
		this.emitter = emitter;
	}

	/** Returns the emitted call text if mi is a recognized JDK intrinsic, else null (the caller
	 * continues with the general Manual/translated-set dispatch). */
	String tryIntrinsic(MethodInvocation mi, IMethodBinding mb) {
		String qualified = mb.getDeclaringClass().getErasure().getQualifiedName();

		// Image.java's own HiDPI provider path (Round 7 gfx) - the only InputStream/OutputStream
		// method this port's final file set reaches through an abstractly-typed receiver.
		if (qualified.equals("java.io.InputStream") && mb.getName().equals("readAllBytes")) {
			emitter.fileImports.add(JRT);
			return "jrt.ReadAllBytes(" + recv(mi) + ")";
		}
		if (qualified.equals("java.lang.Math") && (mb.getName().equals("max") || mb.getName().equals("min"))) {
			return emitMathMinMax(mi, mb);
		}
		if (qualified.equals("java.lang.Math") && mb.getName().equals("abs")) {
			return emitMathAbs(mi, mb);
		}
		// No real Thread/thread-affinity tracking (see Display.thread, an "any" always nil) -
		// checkWidget()'s "!= currentThread()" compares two nils, so it always passes.
		if (qualified.equals("java.lang.Thread") && mb.getName().equals("currentThread")) {
			return "ThreadCurrentThread()";
		}
		// str.substring(start[, end]): byte-slicing, not UTF-16-index-correct for non-ASCII - every
		// call site in the translated set (Widget.getName()) operates on plain class-name ASCII.
		if (qualified.equals("java.lang.String") && mb.getName().equals("substring")) {
			String recv = emitter.expr(mi.getExpression());
			String start = emitter.expr((Expression) mi.arguments().get(0));
			if (mi.arguments().size() == 1) return recv + "[" + start + ":]";
			String end = emitter.expr((Expression) mi.arguments().get(1));
			return recv + "[" + start + ":" + end + "]";
		}
		if (qualified.equals("java.lang.String") && mb.getName().equals("lastIndexOf")) {
			emitter.fileImports.add("strings");
			String recv = emitter.expr(mi.getExpression());
			String ch = emitter.expr((Expression) mi.arguments().get(0));
			return "int32(strings.LastIndexByte(" + recv + ", byte(" + ch + ")))";
		}
		// Integer.toHexString treats its argument as UNSIGNED 32-bit, matching Go's uint32 here.
		if (qualified.equals("java.lang.Integer") && mb.getName().equals("toHexString")) {
			emitter.fileImports.add("strconv");
			String arg = emitter.expr((Expression) mi.arguments().get(0));
			return "strconv.FormatUint(uint64(uint32(" + arg + ")), 16)";
		}
		if (qualified.equals("java.lang.Boolean") && mb.getName().equals("valueOf")) {
			return emitBooleanValueOf(mi, mb);
		}
		// Objects.equals(a, b) on two Go strings is exactly Go's == (only shape used in this set).
		if (qualified.equals("java.util.Objects") && mb.getName().equals("equals") && mi.arguments().size() == 2) {
			Expression arg0 = (Expression) mi.arguments().get(0);
			Expression arg1 = (Expression) mi.arguments().get(1);
			if (isGoStringArg(arg0) && isGoStringArg(arg1)) {
				return "(" + emitter.expr(arg0) + " == " + emitter.expr(arg1) + ")";
			}
		}
		// Every System property key besides "os.arch" reads as unset in this port - matches the
		// real JDK's own behavior for a key that was never set (String.getProperty(...) returns
		// null; Boolean.valueOf(null)/EqualFold("", "true") both come out false either way).
		if (qualified.equals("java.lang.System") && mb.getName().equals("getProperty")) {
			return mi.arguments().size() == 1 ? "\"\"" : arg(mi, 1);
		}
		// str.getChars(0, n, dst, 0): every call site in the translated set copies the whole
		// string, so this only needs to encode str as UTF-16 into dst (see NSString.getChars).
		if (qualified.equals("java.lang.String") && mb.getName().equals("getChars")) {
			emitter.fileImports.add("unicode/utf16");
			String recv = emitter.expr(mi.getExpression());
			String dst = emitter.expr((Expression) mi.arguments().get(2));
			return "copy(" + dst + ", utf16.Encode([]rune(" + recv + ")))";
		}
		if (qualified.equals("java.lang.String") && mb.getName().equals("length")) {
			return "int32(len(" + emitter.expr(mi.getExpression()) + "))";
		}
		if (qualified.equals("java.lang.String") && mb.getName().equals("trim")) {
			emitter.fileImports.add("strings");
			return "strings.TrimSpace(" + emitter.expr(mi.getExpression()) + ")";
		}
		// a.equals(b) on two Go strings is exactly Go's == (Java's String#equals is value equality).
		if (qualified.equals("java.lang.String") && mb.getName().equals("equals")) {
			String recv = emitter.expr(mi.getExpression());
			String arg = emitter.expr((Expression) mi.arguments().get(0));
			return "(" + recv + " == " + arg + ")";
		}
		// System.getProperty("os.arch"): only this one key is needed (IS_X86_64), see manual.txt.
		if (qualified.equals("java.lang.System") && mb.getName().equals("getProperty")
				&& mi.arguments().get(0) instanceof StringLiteral sl && sl.getLiteralValue().equals("os.arch")) {
			return "JavaOsArch()";
		}
		if (qualified.equals("java.lang.System") && mb.getName().equals("arraycopy")) {
			return emitSystemArraycopy(mi);
		}
		// A Consumer<T> is translated as a bare Go func(T) (see GoTypes) - .accept(x) is just
		// calling it, no method to look up.
		if (qualified.equals("java.util.function.Consumer") && mb.getName().equals("accept")) {
			String recv = emitter.expr(mi.getExpression());
			String arg = emitter.expr((Expression) mi.arguments().get(0));
			return recv + "(" + arg + ")";
		}

		// java.lang.Throwable maps to Go's error interface (see Manual) - toString() on it is
		// exactly that interface's own Error() method, not a "ToString" Go doesn't have.
		if (qualified.equals(Manual.JAVA_THROWABLE) && mb.getName().equals("toString")) {
			return emitter.expr(mi.getExpression()) + ".Error()";
		}
		// getClass()/Class#getName(): on Widget's own constructor path (checkSubclass), so it
		// must not degrade to the generic panic marker like other reflection - use reflect.Type.
		if (qualified.equals("java.lang.Object") && mb.getName().equals("getClass") && mi.arguments().isEmpty()) {
			emitter.fileImports.add("reflect");
			String recv = mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
			return "reflect.TypeOf(" + recv + ")";
		}
		if (qualified.equals("java.lang.Class") && mb.getName().equals("getName")) {
			return emitter.expr(mi.getExpression()) + ".String()";
		}
		return tryMore(mi, mb, qualified, mb.getName());
	}

	private static final String JRT = "github.com/haiodo/gowt/internal/jrt";

	private String arg(MethodInvocation mi, int i) {
		return emitter.expr((Expression) mi.arguments().get(i));
	}

	private String recv(MethodInvocation mi) {
		return mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
	}

	/** Round 6 additions: boxing, monitors, a few String/Math/System members, the Selector enum. */
	private String tryMore(MethodInvocation mi, IMethodBinding mb, String qualified, String name) {
		String key = qualified + "#" + name;
		switch (key) {
			// Selector is elided (README): Selector.valueOf(sel) is just sel, its constants OS's sel_x.
			case "org.eclipse.swt.internal.cocoa.Selector#valueOf":
				return arg(mi, 0);
			case "java.lang.Math#ceil", "java.lang.Math#floor":
				emitter.fileImports.add("math");
				return "math." + (name.equals("ceil") ? "Ceil" : "Floor") + "(float64(" + arg(mi, 0) + "))";
			// Java rounds half up: floor(x + 0.5), int for a float argument, long for a double.
			case "java.lang.Math#round":
				emitter.fileImports.add("math");
				return (mb.getParameterTypes()[0].getName().equals("float") ? "int32" : "int64")
						+ "(math.Floor(float64(" + arg(mi, 0) + ") + 0.5))";
			case "java.lang.Math#hypot":
				emitter.fileImports.add("math");
				return "math.Hypot(" + arg(mi, 0) + ", " + arg(mi, 1) + ")";
			case "java.lang.Float#floatToIntBits":
				emitter.fileImports.add("math");
				return "int32(math.Float32bits(" + arg(mi, 0) + "))";
			// WinBMPFileFormat's BI_BITFIELDS mask conversion (LEDataInputStream is little-endian,
			// ImageData masks are expected big-endian for depth != 16 - see its own comment).
			case "java.lang.Integer#reverseBytes":
				emitter.fileImports.add("math/bits");
				return "int32(bits.ReverseBytes32(uint32(" + arg(mi, 0) + ")))";
			// Boxed Integer is Go any (Manual); unboxing asserts back, nil reads as 0.
			case "java.lang.Integer#valueOf":
				return mb.getParameterTypes()[0].isPrimitive() ? arg(mi, 0) : null;
			case "java.lang.Integer#intValue":
				emitter.fileImports.add(JRT);
				return "jrt.Cast[int32](" + recv(mi) + ")";
			case "java.lang.Boolean#booleanValue":
				return recv(mi);
			case "java.util.Objects#requireNonNull", "java.util.Objects#nonNull":
				return name.equals("nonNull") ? "(" + arg(mi, 0) + " != nil)" : arg(mi, 0);
			// Go package init already ran every static initializer Class.forName would force.
			case "java.lang.Class#forName":
				emitter.fileImports.add(JRT);
				return "jrt.ClassForName(" + arg(mi, 0) + ")";
			case "java.lang.Throwable#printStackTrace":
				emitter.fileImports.add("fmt");
				emitter.fileImports.add("os");
				return "fmt.Fprintln(os.Stderr, " + recv(mi) + ")";
			case "java.lang.Throwable#getMessage":
				return recv(mi) + ".Error()";
			case "java.lang.Throwable#getCause":
				emitter.fileImports.add("errors");
				return "errors.Unwrap(" + recv(mi) + ")";
			case "java.lang.Boolean#parseBoolean":
				emitter.fileImports.add("strings");
				return "strings.EqualFold(" + arg(mi, 0) + ", \"true\")";
			// No Java system properties in this port (see getProperty above).
			case "java.lang.Boolean#getBoolean":
				return "false";
			case "java.lang.String#equalsIgnoreCase":
				emitter.fileImports.add("strings");
				return "strings.EqualFold(" + recv(mi) + ", " + arg(mi, 0) + ")";
			case "java.lang.String#indexOf":
				String needle = mb.getParameterTypes()[0].isPrimitive() ? "string(rune(" + arg(mi, 0) + "))" : arg(mi, 0);
				if (mi.arguments().size() == 2) {
					emitter.fileImports.add(JRT);
					return "jrt.IndexFrom(" + recv(mi) + ", " + needle + ", " + arg(mi, 1) + ")";
				}
				emitter.fileImports.add("strings");
				return "int32(strings.Index(" + recv(mi) + ", " + needle + "))";
			case "java.lang.String#isEmpty":
				return "(len(" + recv(mi) + ") == 0)";
			case "java.lang.Integer#parseInt":
				emitter.fileImports.add(JRT);
				return mi.arguments().size() == 1 ? "jrt.ParseInt(" + arg(mi, 0) + ")" : null;
			case "java.lang.Integer#toString", "java.lang.Boolean#toString":
				if (mi.arguments().size() != 1 || mi.getExpression() == null) return null;
				emitter.fileImports.add("fmt");
				return "fmt.Sprint(" + arg(mi, 0) + ")";
			// One resource registry per process (jrt.RegisterResources), not per class loader.
			case "java.lang.Class#getResourceAsStream":
				emitter.fileImports.add(JRT);
				return "jrt.ClassGetResourceAsStream(" + recv(mi) + ", " + arg(mi, 0) + ")";
			case "java.lang.String#charAt":
				emitter.fileImports.add("unicode/utf16");
				return "utf16.Encode([]rune(" + recv(mi) + "))[" + arg(mi, 0) + "]";
			case "java.lang.String#valueOf":
				return stringValueOf(mi, mb);
			case "java.lang.System#nanoTime":
				emitter.fileImports.add("time");
				return "time.Now().UnixNano()";
			case "java.io.PrintStream#println":
				return println(mi);
			// Object monitors: the one global jrt monitor (see ControlFlowEmitter.emitSynchronized).
			case "java.lang.Object#wait":
				emitter.fileImports.add(JRT);
				return "jrt.MonitorWait()";
			case "java.lang.Object#notifyAll", "java.lang.Object#notify":
				emitter.fileImports.add(JRT);
				return "jrt.MonitorNotifyAll()";
			// Locale is only ever asked for its language (Display's nib lookup): $LANG's, else "en".
			case "java.util.Locale#getDefault":
				return "any(nil)";
			case "java.util.Locale#getLanguage":
				emitter.fileImports.add(JRT);
				return "jrt.LocaleLanguage(" + recv(mi) + ")";
			// Display.getAwtRunLoopMode's JDK-version check for AWT embedding: not a JVM, so
			// version 0 - no AWT run loop mode.
			case "java.lang.Runtime#version":
				return "any(nil)";
			case "java.lang.Runtime.Version#feature":
				emitter.fileImports.add(JRT);
				return "jrt.JavaVersionFeature(" + recv(mi) + ")";
			// Resource's leak tracker (off unless enabled): no Cleaner in Go, the tracker stays nil.
			case "java.lang.ref.Cleaner#create":
				return "any(nil)";
			// Go has no shutdown hooks; the argument (an anonymous Thread) is not evaluated.
			case "java.lang.Runtime#addShutdownHook":
				return "/* Runtime.addShutdownHook dropped */";
			default:
				return null;
		}
	}

	private String stringValueOf(MethodInvocation mi, IMethodBinding mb) {
		ITypeBinding p = mb.getParameterTypes()[0];
		String a = arg(mi, 0);
		if (p.getName().equals("char")) return "string(rune(" + a + "))";
		if (p.isArray() && p.getComponentType().getName().equals("char")) {
			emitter.fileImports.add("unicode/utf16");
			return "string(utf16.Decode(" + a + "))";
		}
		emitter.fileImports.add("fmt");
		return "fmt.Sprint(" + a + ")";
	}

	/** System.out/err.println(x): the receiver is the only thing telling stdout from stderr. */
	private String println(MethodInvocation mi) {
		if (!(mi.getExpression() instanceof QualifiedName qn)) return null;
		emitter.fileImports.add("fmt");
		emitter.fileImports.add("os");
		String stream = qn.getName().getIdentifier().equals("err") ? "os.Stderr" : "os.Stdout";
		String a = mi.arguments().isEmpty() ? "" : ", " + arg(mi, 0);
		return "fmt.Fprintln(" + stream + a + ")";
	}

	/**
	 * java.lang.Math has no translated-set/manual rule of its own; Math.max/min(float,float) is
	 * the only overload the 4 target files use, so it is mapped inline to Go's math package.
	 */
	private String emitMathMinMax(MethodInvocation mi, IMethodBinding mb) {
		emitter.fileImports.add("math");
		String goFn = mb.getName().equals("max") ? "math.Max" : "math.Min";
		String a = emitter.expr((Expression) mi.arguments().get(0));
		String b = emitter.expr((Expression) mi.arguments().get(1));
		String goReturn = dev.gowt.j2go.GoTypes.map(mb.getReturnType(), emitter);
		return goReturn + "(" + goFn + "(float64(" + a + "), float64(" + b + ")))";
	}

	private boolean isGoStringArg(Expression e) {
		org.eclipse.jdt.core.dom.ITypeBinding t = e.resolveTypeBinding();
		return t != null && dev.gowt.j2go.GoTypes.map(t, emitter).equals("string");
	}

	/** Math.abs(int/long/float/double): math.Abs for floats, a manual branch for integers (Go's
	 * math package only has a float64 Abs). */
	private String emitMathAbs(MethodInvocation mi, IMethodBinding mb) {
		String arg = emitter.expr((Expression) mi.arguments().get(0));
		String goReturn = dev.gowt.j2go.GoTypes.map(mb.getReturnType(), emitter);
		if (goReturn.equals("float32") || goReturn.equals("float64")) {
			emitter.fileImports.add("math");
			return goReturn + "(math.Abs(float64(" + arg + ")))";
		}
		String tmp = "abs" + (++emitter.tempCounter);
		emitter.prelude.add(tmp + " := " + arg);
		emitter.prelude.add("if " + tmp + " < 0 { " + tmp + " = -" + tmp + " }");
		return tmp;
	}

	/** Boolean.valueOf(boolean) is a no-op box; Boolean.valueOf(String) parses case-insensitively. */
	private String emitBooleanValueOf(MethodInvocation mi, IMethodBinding mb) {
		String arg = emitter.expr((Expression) mi.arguments().get(0));
		if (dev.gowt.j2go.GoTypes.map(mb.getParameterTypes()[0], emitter).equals("bool")) return arg;
		emitter.fileImports.add("strings");
		return "strings.EqualFold(" + arg + ", \"true\")";
	}

	/** System.arraycopy(src,srcPos,dst,dstPos,len) -> Go's copy() over the matching sub-slices. */
	private String emitSystemArraycopy(MethodInvocation mi) {
		java.util.List<?> a = mi.arguments();
		String src = emitter.expr((Expression) a.get(0));
		String srcPos = emitter.expr((Expression) a.get(1));
		String dst = emitter.expr((Expression) a.get(2));
		String dstPos = emitter.expr((Expression) a.get(3));
		String length = emitter.expr((Expression) a.get(4));
		return "copy(" + dst + "[" + dstPos + ":], " + src + "[" + srcPos + ":" + srcPos + "+" + length + "])";
	}
}
