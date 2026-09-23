package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import org.eclipse.jdt.core.dom.Expression;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.MethodInvocation;
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
		if (qualified.equals("java.lang.System") && mb.getName().equals("getProperty") && mi.arguments().size() == 1) {
			return "\"\"";
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
		return null;
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
