package dev.gowt.j2go.emit;

import org.eclipse.jdt.core.dom.Expression;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.ITypeBinding;
import org.eclipse.jdt.core.dom.MethodInvocation;

import static dev.gowt.j2go.emit.JdkIntrinsics.JRT;

/** More JDK members (arrays, String, Float, Byte) the gtk sources use; JdkCalls falls through to here. */
final class JdkCallsExtra {

	private JdkCallsExtra() {}

	private static String arg(Emitter e, MethodInvocation mi, int i) {
		return e.expr((Expression) mi.arguments().get(i));
	}

	private static String recv(Emitter e, MethodInvocation mi) {
		return mi.getExpression() != null ? e.expr(mi.getExpression()) : "this";
	}

	static String tryMore(Emitter e, MethodInvocation mi, IMethodBinding mb, String qualified, String name) {
		switch (qualified + "#" + name) {
			// No JVM: a shutdown hook is never run, so none is registered or removed.
			case "java.lang.Runtime#getRuntime":
				return "any(nil)";
			case "java.lang.Runtime#removeShutdownHook":
				return "false";
			// Streams are eager *jrt.List values; Collect returns any, so the Java result type is asserted back.
			case "java.util.stream.Collectors#joining":
				e.fileImports.add(JRT);
				return "jrt.CollectorsJoining(" + arg(e, mi, 0) + ")";
			case "java.util.stream.Collectors#toList", "java.util.stream.Collectors#toSet":
				e.fileImports.add(JRT);
				return "jrt.CollectorsTo" + (name.equals("toList") ? "List" : "Set") + "()";
			case "java.util.stream.Stream#collect": {
				e.fileImports.add(JRT);
				String t = dev.gowt.j2go.GoTypes.map(mi.resolveTypeBinding(), e);
				return recv(e, mi) + ".Collect(" + arg(e, mi, 0) + ")" + (t.equals("any") ? "" : ".(" + t + ")");
			}
			case "java.util.Arrays#stream":
				if (mi.arguments().size() != 1) return null;
				e.fileImports.add(JRT);
				return "jrt.ArraysAsList(" + arg(e, mi, 0) + ")";
			case "java.lang.String#repeat":
				e.fileImports.add("strings");
				return "strings.Repeat(" + recv(e, mi) + ", int(" + arg(e, mi, 0) + "))";
			case "java.lang.Character#getDirectionality":
				e.fileImports.add(JRT);
				return "jrt.Directionality(rune(" + arg(e, mi, 0) + "))";
			case "java.lang.Character#isLetterOrDigit":
				e.fileImports.add("unicode");
				return "(unicode.IsLetter(rune(" + arg(e, mi, 0) + ")) || unicode.IsDigit(rune(" + arg(e, mi, 0) + ")))";
			case "java.lang.Character#isHighSurrogate":
				e.fileImports.add(JRT);
				return "jrt.IsHighSurrogate(rune(" + arg(e, mi, 0) + "))";
			case "java.lang.String#compareTo":
				e.fileImports.add(JRT);
				return "jrt.StringCompareTo(" + recv(e, mi) + ", " + arg(e, mi, 0) + ")";
			case "java.lang.Byte#toUnsignedInt":
				return "int32(uint8(" + arg(e, mi, 0) + "))";
			case "java.util.Arrays#fill":
				e.fileImports.add(JRT);
				return mi.arguments().size() == 2 ? "jrt.ArraysFill(" + arg(e, mi, 0) + ", " + arg(e, mi, 1) + ")"
						: "jrt.ArraysFillRange(" + arg(e, mi, 0) + ", " + arg(e, mi, 1) + ", " + arg(e, mi, 2) + ", " + arg(e, mi, 3) + ")";
			case "java.util.Arrays#equals":
				e.fileImports.add("slices");
				return "slices.Equal(" + arg(e, mi, 0) + ", " + arg(e, mi, 1) + ")";
			case "java.lang.Object#clone": {
				ITypeBinding t = mi.getExpression() == null ? null : mi.getExpression().resolveTypeBinding();
				if (t == null || !t.isArray()) return null;
				e.fileImports.add("slices");
				return "slices.Clone(" + recv(e, mi) + ")";
			}
			case "java.lang.String#toCharArray":
				e.fileImports.add("unicode/utf16");
				return "utf16.Encode([]rune(" + recv(e, mi) + "))";
			case "java.lang.String#concat":
				return "(" + recv(e, mi) + " + " + arg(e, mi, 0) + ")";
			case "java.lang.String#replaceFirst", "java.lang.String#replaceAll":
				e.fileImports.add(JRT);
				return "jrt.Replace" + (name.equals("replaceAll") ? "All" : "First") + "(" + recv(e, mi) + ", " + arg(e, mi, 0) + ", " + arg(e, mi, 1) + ")";
			case "java.lang.Float#isNaN":
				e.fileImports.add("math");
				return "math.IsNaN(float64(" + arg(e, mi, 0) + "))";
			case "java.lang.Float#isInfinite":
				e.fileImports.add("math");
				return "math.IsInf(float64(" + arg(e, mi, 0) + "), 0)";
			case "java.io.PrintStream#print", "java.io.PrintStream#format", "java.io.PrintStream#printf": {
				if (mi.getExpression() == null) return null;
				e.fileImports.add("fmt");
				String stream;
				if (mi.getExpression() instanceof org.eclipse.jdt.core.dom.QualifiedName qn && qn.getQualifier().getFullyQualifiedName().equals("System")) {
					e.fileImports.add("os");
					stream = qn.getName().getIdentifier().equals("err") ? "os.Stderr" : "os.Stdout";
				} else {
					e.fileImports.add("io");
					stream = recv(e, mi) + ".(io.Writer)";
				}
				if (name.equals("print")) return "fmt.Fprint(" + stream + ", " + arg(e, mi, 0) + ")";
				e.fileImports.add(JRT);
				java.util.List<String> rest = new java.util.ArrayList<>();
				for (int i = 1; i < mi.arguments().size(); i++) rest.add(arg(e, mi, i));
				return "fmt.Fprint(" + stream + ", jrt.Format(" + arg(e, mi, 0) + ", []any{" + String.join(", ", rest) + "}))";
			}
			default:
				return null;
		}
	}
}
