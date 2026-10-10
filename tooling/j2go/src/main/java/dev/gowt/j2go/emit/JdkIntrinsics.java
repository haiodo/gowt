package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.Expression;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.ITypeBinding;
import org.eclipse.jdt.core.dom.MethodInvocation;
import org.eclipse.jdt.core.dom.QualifiedName;
import org.eclipse.jdt.core.dom.StringLiteral;

import java.util.List;

/** Mapping of plain JDK calls used by the translated set: Math.min/max, System.arraycopy,
 * System.getProperty("os.arch"), String/Consumer/Throwable methods. The single place to add
 * a new JDK method mapping. */
final class JdkIntrinsics {

	private final Emitter emitter;

	// String methods that are one strings.X(receiver, args...) call, by Java name and arity.
	private static final java.util.Map<String, String> STRING_FUNCS = java.util.Map.of(
			"startsWith/1", "strings.HasPrefix", "endsWith/1", "strings.HasSuffix", "contains/1", "strings.Contains",
			"toLowerCase/0", "strings.ToLower", "toUpperCase/0", "strings.ToUpper");

	private final JdkCalls more;

	JdkIntrinsics(Emitter emitter) {
		this.emitter = emitter;
		this.more = new JdkCalls(emitter);
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
		// Thread identity is the goroutine id (internal/jrt/lang.go), shared by swt and the tests.
		if (qualified.equals("java.lang.Thread") && mb.getName().equals("currentThread")) {
			return jrtCall("CurrentThread", "");
		}
		if (qualified.equals("java.lang.String") && mb.getName().equals("substring")) {
			return jrtCall("Substring", recv(mi) + ", " + arg(mi, 0) + ", " + (mi.arguments().size() == 1 ? "-1" : arg(mi, 1)));
		}
		if (qualified.equals("java.lang.String") && mb.getName().equals("lastIndexOf")) {
			boolean strArg = ((Expression) mi.arguments().get(0)).resolveTypeBinding().getQualifiedName().equals("java.lang.String");
			return jrtCall("LastIndexOf", recv(mi) + ", " + (strArg ? arg(mi, 0) : "string(rune(" + arg(mi, 0) + "))"));
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
			// Any other operand: null-safe a.equals(b) (ToolItem.setBackground compares Colors).
			return jrtCall("ObjectsEquals", emitter.expr(arg0) + ", " + emitter.expr(arg1));
		}
		// Every System property key besides "os.arch", "os.name" and "line.separator" (jrt.SystemProperties) reads as unset in this port - matches the
		// real JDK's own behavior for a key that was never set (String.getProperty(...) returns
		// null; Boolean.valueOf(null)/EqualFold("", "true") both come out false either way).
		if (qualified.equals("java.lang.System") && mb.getName().equals("getProperty")) {
			return jrtCall("GetProperty", arg(mi, 0) + ", " + (mi.arguments().size() == 1 ? "\"\"" : arg(mi, 1)));
		}
		if (qualified.equals("java.lang.String") && mb.getName().equals("getChars")) {
			List<Expression> a = mi.arguments();
			return jrtCall("GetChars", emitter.expr(mi.getExpression()) + ", " + emitter.expr(a.get(0)) + ", " + emitter.expr(a.get(1)) + ", "
					+ emitter.expr(a.get(2)) + ", " + emitter.expr(a.get(3)));
		}
		String stringFunc = STRING_FUNCS.get(mb.getName() + "/" + mi.arguments().size());
		if (qualified.equals("java.lang.String") && stringFunc != null) {
			emitter.fileImports.add("strings");
			return stringFunc + "(" + emitter.expr(mi.getExpression()) + (mi.arguments().isEmpty() ? "" : ", " + arg(mi, 0)) + ")";
		}
		// hashCode surface of the value types' own hashCode() (FontData, FontMetrics, Image).
		if (qualified.equals("java.lang.String") && mb.getName().equals("hashCode")) return jrtCall("StringHashCode", recv(mi));
		if (qualified.equals("java.lang.Double") && mb.getName().equals("hashCode") && mi.arguments().size() == 1) return jrtCall("DoubleHashCode", arg(mi, 0));
		if (qualified.equals("java.util.Objects") && mb.getName().equals("hash")) {
			List<String> args = new java.util.ArrayList<>();
			for (int i = 0; i < mi.arguments().size(); i++) args.add(arg(mi, i));
			return jrtCall("ObjectsHash", String.join(", ", args));
		}
		if (qualified.equals("java.lang.System") && mb.getName().equals("lineSeparator")) return jrtCall("LineSeparator", "");
		if (qualified.equals("java.lang.String") && mb.getName().equals("length")) {
			return jrtCall("StringLength", emitter.expr(mi.getExpression()));
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

		// Function/Supplier/Comparator/...: a bare Go func (see GoTypes.isJdkFunctional), its one abstract method is the call.
		if (dev.gowt.j2go.GoTypes.isJdkFunctional(qualified) && java.lang.reflect.Modifier.isAbstract(mb.getModifiers())) {
			return recv(mi) + "(" + String.join(", ", emitter.buildArgs(mi.arguments(), mb)) + ")";
		}

		// java.lang.Throwable maps to Go's error interface (see Manual) - toString() on it is
		// exactly that interface's own Error() method, not a "ToString" Go doesn't have.
		if (qualified.equals(Manual.JAVA_THROWABLE) && mb.getName().equals("toString")) {
			return emitter.expr(mi.getExpression()) + ".Error()";
		}
		// getClass()/Class#getName(): on Widget's own constructor path (checkSubclass), so it
		// must not degrade to the generic panic marker like other reflection - use reflect.Type.
		// A cascade class's receiver expression is frequently an upcast promoted-field address
		// (Widget[] widgets = getExampleWidgets() stores &concreteText.Widget) - reflect.TypeOf on
		// that gives the upcast type (*Widget), not the real dynamic one Java's getClass() means.
		// This class's own .impl (Round 6+ impl cascade) always holds the true concrete pointer,
		// so route through .Impl() instead whenever the receiver's static type has one.
		if (qualified.equals("java.lang.Object") && mb.getName().equals("getClass") && mi.arguments().isEmpty()) {
			emitter.fileImports.add("reflect");
			String recv = mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
			ITypeBinding receiverType = mi.getExpression() != null ? mi.getExpression().resolveTypeBinding() : null;
			TypeModel.ClassInfo rci = receiverType != null ? emitter.model.lookup(receiverType) : emitter.currentClassInfo;
			if (hasImpl(rci)) return "reflect.TypeOf(" + recv + ".Impl())";
			return "reflect.TypeOf(" + recv + ")";
		}
		// Object.equals with no override: Java's default is reference identity, same embedded-
		// pointer-level problem as getClass() above - see emitObjectEquals.
		if (qualified.equals("java.lang.Object") && mb.getName().equals("equals") && mi.arguments().size() == 1) {
			return emitObjectEquals(mi);
		}
		// Round 10 reflection: Class<?> stays reflect.Type (Manual) - see README "Round 10
		// reflection" and internal/jrt/reflect.go.
		// Class<F>.cast(x): erased generics keep x as is; the caller's call-site narrowing reads it back as F.
		if (qualified.equals("java.lang.Class") && mb.getName().equals("cast")) {
			Expression a = (Expression) mi.arguments().get(0);
			return emitter.upcastObject(emitter.expr(a), a.resolveTypeBinding(), mi.resolveTypeBinding());
		}
		if (qualified.equals("java.lang.Class") && mb.getName().equals("getName")) {
			emitter.fileImports.add(JRT);
			return "jrt.ClassName(" + recv(mi) + ")";
		}
		if (qualified.equals("java.lang.Class") && mb.getName().equals("isArray")) {
			emitter.fileImports.add("reflect");
			return recv(mi) + ".Kind() == reflect.Slice";
		}
		if (qualified.equals("java.lang.Class") && mb.getName().equals("getComponentType")) {
			return recv(mi) + ".Elem()";
		}
		if (qualified.equals("java.lang.Class") && mb.getName().equals("getDeclaredField")) {
			emitter.fileImports.add(JRT);
			return "jrt.ClassGetDeclaredField(" + recv(mi) + ", " + arg(mi, 0) + ")";
		}
		if ((qualified.equals("java.lang.reflect.Field") || qualified.equals("java.lang.reflect.Method")) && mb.getName().equals("setAccessible")) {
			return "func() any { return nil }()";
		}
		if (qualified.equals("java.lang.reflect.Field") && mb.getName().equals("get")) {
			ITypeBinding targetType = ((Expression) mi.arguments().get(0)).resolveTypeBinding();
			String target = targetType != null && hasImpl(emitter.model.lookup(targetType)) ? arg(mi, 0) + ".Impl()" : arg(mi, 0);
			return recv(mi) + ".(*jrt.Field).Get(" + target + ")";
		}
		if (qualified.equals("java.lang.Class") && (mb.getName().equals("getMethod") || mb.getName().equals("getDeclaredMethod"))) {
			List<String> args = emitter.buildArgs(mi.arguments(), mb);
			emitter.fileImports.add(JRT);
			return "jrt.ClassGetMethod(" + recv(mi) + ", " + args.get(0) + ", " + args.get(1) + ")";
		}
		if (qualified.equals("java.lang.reflect.Method") && mb.getName().equals("invoke")) {
			List<String> args = emitter.buildArgs(mi.arguments(), mb);
			emitter.fileImports.add(JRT);
			// Same .Impl() rule as getClass: the registered closure needs the concrete object.
			ITypeBinding targetType = ((Expression) mi.arguments().get(0)).resolveTypeBinding();
			String target = targetType != null && hasImpl(emitter.model.lookup(targetType)) ? args.get(0) + ".Impl()" : args.get(0);
			return recv(mi) + ".(*jrt.Method).Invoke(" + target + ", " + args.get(1) + "...)";
		}
		if (qualified.equals("java.lang.reflect.Method") && mb.getName().equals("getName")) {
			return recv(mi) + ".(*jrt.Method).GetName()";
		}
		if (qualified.equals("java.lang.reflect.Method") && mb.getName().equals("getReturnType")) {
			return recv(mi) + ".(*jrt.Method).GetReturnType()";
		}
		// Object.toString() on an Object-typed receiver (Method.invoke's boxed result): fmt.Sprint
		// picks up every translated class's own Go String() method via fmt.Stringer.
		if (qualified.equals("java.lang.Object") && mb.getName().equals("toString") && mi.arguments().isEmpty()) {
			emitter.fileImports.add("fmt");
			return "fmt.Sprint(" + recv(mi) + ")";
		}
		return more.tryMore(mi, mb, qualified, mb.getName());
	}

	static final String JRT = "github.com/haiodo/gowt/internal/jrt";

	private String jrtCall(String fn, String args) {
		emitter.fileImports.add(JRT);
		return "jrt." + fn + "(" + args + ")";
	}

	private String arg(MethodInvocation mi, int i) {
		return emitter.expr((Expression) mi.arguments().get(i));
	}

	private String recv(MethodInvocation mi) {
		return mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
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

	private static boolean hasImpl(TypeModel.ClassInfo ci) {
		return ci != null && ci.root.splitsDispatch() && ci.root.hasImpl();
	}

	// A translated object can be referenced through different embedded-pointer levels
	// (&shell.Widget vs shell): .Impl() normalizes to the concrete leaf pointer instead.
	private String emitObjectEquals(MethodInvocation mi) {
		Expression argExpr = (Expression) mi.arguments().get(0);
		String recv = recv(mi);
		String arg = emitter.expr(argExpr);
		ITypeBinding receiverType = mi.getExpression() != null ? mi.getExpression().resolveTypeBinding() : null;
		TypeModel.ClassInfo rci = receiverType != null ? emitter.model.lookup(receiverType) : emitter.currentClassInfo;
		TypeModel.ClassInfo aci = argExpr.resolveTypeBinding() != null ? emitter.model.lookup(argExpr.resolveTypeBinding()) : null;
		boolean rHas = hasImpl(rci);
		boolean aHas = hasImpl(aci);
		if (!rHas && !aHas) return "(" + recv + " == " + arg + ")";
		String l = rHas ? recv + ".Impl()" : recv;
		String r = aHas ? arg + ".Impl()" : arg;
		return "(any(" + l + ") == any(" + r + "))";
	}
}
