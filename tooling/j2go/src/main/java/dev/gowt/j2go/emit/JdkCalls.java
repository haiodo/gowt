package dev.gowt.j2go.emit;

import org.eclipse.jdt.core.dom.Expression;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.ITypeBinding;
import org.eclipse.jdt.core.dom.MethodInvocation;
import org.eclipse.jdt.core.dom.QualifiedName;

import static dev.gowt.j2go.emit.JdkIntrinsics.JRT;

/** Table of the remaining JDK members, dispatched by "Class#method" (see JdkIntrinsics.tryIntrinsic). */
final class JdkCalls {

	private final Emitter emitter;

	JdkCalls(Emitter emitter) {
		this.emitter = emitter;
	}

	private String arg(MethodInvocation mi, int i) {
		return emitter.expr((Expression) mi.arguments().get(i));
	}

	private String recv(MethodInvocation mi) {
		return mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
	}

	/** boxing, monitors, a few String/Math/System members, the Selector enum. */
	String tryMore(MethodInvocation mi, IMethodBinding mb, String qualified, String name) {
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
			case "java.lang.Math#log", "java.lang.Math#sqrt", "java.lang.Math#exp", "java.lang.Math#sin", "java.lang.Math#cos":
				emitter.fileImports.add("math");
				return "math." + dev.gowt.j2go.Names.capitalize(name) + "(float64(" + arg(mi, 0) + "))";
			case "java.lang.Math#pow":
				emitter.fileImports.add("math");
				return "math.Pow(float64(" + arg(mi, 0) + "), float64(" + arg(mi, 1) + "))";
			case "java.lang.System#gc":
				emitter.fileImports.add(JRT);
				return "jrt.GC()";
			case "java.lang.System#setProperty":
				emitter.fileImports.add(JRT);
				return "jrt.SetProperty(" + arg(mi, 0) + ", " + arg(mi, 1) + ")";
			case "java.lang.System#getProperties":
				emitter.fileImports.add(JRT);
				return "jrt.SystemProperties";
			case "java.lang.System#getenv":
				if (mi.arguments().size() != 1) return null;
				emitter.fileImports.add(JRT);
				return "jrt.Getenv(" + arg(mi, 0) + ")";
			case "java.lang.String#isBlank":
				emitter.fileImports.add("strings");
				return "(strings.TrimSpace(" + recv(mi) + ") == \"\")";
			case "java.lang.String#matches":
				emitter.fileImports.add(JRT);
				return "jrt.Matches(" + recv(mi) + ", " + arg(mi, 0) + ")";
			case "java.lang.Math#hypot":
				emitter.fileImports.add("math");
				String[] h = new String[2];
				for (int i = 0; i < 2; i++) h[i] = ((Expression) mi.arguments().get(i)).resolveTypeBinding().getName().equals("float") ? "float64(" + arg(mi, i) + ")" : arg(mi, i);
				return "math.Hypot(" + h[0] + ", " + h[1] + ")";
			case "java.lang.Float#floatToIntBits":
				emitter.fileImports.add("math");
				return "int32(math.Float32bits(" + arg(mi, 0) + "))";
			// WinBMPFileFormat's BI_BITFIELDS mask conversion (LEDataInputStream is little-endian,
			// ImageData masks are expected big-endian for depth != 16 - see its own comment).
			case "java.lang.Integer#reverseBytes":
				emitter.fileImports.add("math/bits");
				return "int32(bits.ReverseBytes32(uint32(" + arg(mi, 0) + ")))";
			// Boxed Integer is Go any (Manual); unboxing asserts back, nil reads as 0. The
			// String overload (Set/Get dialog's numeric parameters) parses instead of boxing.
			case "java.lang.Integer#valueOf":
				if (mb.getParameterTypes()[0].isPrimitive()) return arg(mi, 0);
				emitter.fileImports.add(JRT);
				return "jrt.ParseInt(" + arg(mi, 0) + ")";
			case "java.lang.Long#valueOf":
				if (mb.getParameterTypes()[0].isPrimitive()) return arg(mi, 0);
				emitter.fileImports.add(JRT);
				return "jrt.ParseLong(" + arg(mi, 0) + ")";
			// Character has no String-parsing overload - always a primitive-char box (no-op).
			case "java.lang.Character#valueOf":
				return arg(mi, 0);
			case "java.lang.Integer#intValue":
				emitter.fileImports.add(JRT);
				return "jrt.Cast[int32](" + recv(mi) + ")";
			case "java.lang.reflect.Array#getLength":
				emitter.fileImports.add("reflect");
				return "int32(reflect.ValueOf(" + arg(mi, 0) + ").Len())";
			case "java.lang.reflect.Array#get":
				emitter.fileImports.add("reflect");
				return "reflect.ValueOf(" + arg(mi, 0) + ").Index(int(" + arg(mi, 1) + ")).Interface()";
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
			// Only the array-argument form; the elements-as-varargs form stays unresolved.
			case "java.util.Arrays#asList":
				if (mi.arguments().size() != 1 || !((Expression) mi.arguments().get(0)).resolveTypeBinding().isArray()) return null;
				emitter.fileImports.add(JRT);
				return "jrt.ArraysAsList(" + arg(mi, 0) + ")";
			case "java.lang.String#toString":
				return recv(mi);
			// char is uint16 (a code point when the parameter is int); Java's isWhitespace/isDigit are close to unicode's.
			case "java.lang.Character#isDigit", "java.lang.Character#isWhitespace", "java.lang.Character#isLetter":
				emitter.fileImports.add("unicode");
				return "unicode.Is" + (name.equals("isWhitespace") ? "Space" : dev.gowt.j2go.Names.capitalize(name).substring(2)) + "(rune(" + arg(mi, 0) + "))";
			case "java.lang.Character#toLowerCase", "java.lang.Character#toUpperCase":
				emitter.fileImports.add("unicode");
				return dev.gowt.j2go.GoTypes.map(mb.getReturnType(), emitter) + "(unicode.To" + (name.equals("toLowerCase") ? "Lower" : "Upper") + "(rune(" + arg(mi, 0) + ")))";
			case "java.lang.Double#compare":
				emitter.fileImports.add(JRT);
				return "jrt.DoubleCompare(float64(" + arg(mi, 0) + "), float64(" + arg(mi, 1) + "))";
			case "java.lang.Double#toString":
				if (mi.arguments().size() != 1) return null;
				emitter.fileImports.add(JRT);
				return "jrt.DoubleToString(float64(" + arg(mi, 0) + "))";
			case "java.lang.String#format":
				if (!mb.getParameterTypes()[0].getQualifiedName().equals("java.lang.String")) return null;
				emitter.fileImports.add(JRT);
				java.util.List<String> rest = new java.util.ArrayList<>();
				for (int i = 1; i < mi.arguments().size(); i++) rest.add(arg(mi, i));
				return "jrt.Format(" + arg(mi, 0) + ", []any{" + String.join(", ", rest) + "})";
			case "java.lang.String#replace": {
				emitter.fileImports.add("strings");
				String[] r = new String[2];
				for (int i = 0; i < 2; i++) r[i] = mb.getParameterTypes()[i].isPrimitive() ? "string(rune(" + arg(mi, i) + "))" : arg(mi, i);
				return "strings.ReplaceAll(" + recv(mi) + ", " + r[0] + ", " + r[1] + ")";
			}
			case "java.util.concurrent.ConcurrentHashMap#newKeySet":
				emitter.fileImports.add(JRT);
				return "jrt.NewList()";
			case "java.util.Arrays#fill":
				if (mi.arguments().size() != 2) return null;
				emitter.fileImports.add(JRT);
				return "jrt.Fill(" + arg(mi, 0) + ", " + arg(mi, 1) + ")";
			case "java.lang.String#toCharArray":
				emitter.fileImports.add("unicode/utf16");
				return "utf16.Encode([]rune(" + recv(mi) + "))";
			case "java.util.Arrays#copyOf":
				emitter.fileImports.add(JRT);
				return "jrt.CopyOf(" + arg(mi, 0) + ", " + arg(mi, 1) + ")";
			case "java.lang.String#split":
				emitter.fileImports.add(JRT);
				return "jrt.Split(" + recv(mi) + ", " + arg(mi, 0) + ")";
			case "java.lang.Float#parseFloat":
				emitter.fileImports.add(JRT);
				return "jrt.ParseFloat(" + arg(mi, 0) + ")";
			// Object.hashCode() through an interface- or Object-typed receiver: the receiver's own HashCode() if it has one.
			case "java.lang.Object#hashCode":
				emitter.fileImports.add(JRT);
				return "jrt.HashCodeOf(" + recv(mi) + ")";
			case "java.util.Comparator#comparingInt":
				emitter.fileImports.add(JRT);
				return "jrt.ComparingInt(" + arg(mi, 0) + ")";
			case "java.util.Comparator#thenComparing":
				emitter.fileImports.add(JRT);
				return "jrt.ThenComparing(" + recv(mi) + ", " + arg(mi, 0) + ")";
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
				emitter.fileImports.add(JRT);
				return "jrt.IndexFrom(" + recv(mi) + ", " + needle + ", " + (mi.arguments().size() == 2 ? arg(mi, 1) : "0") + ")";
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
			case "java.lang.Thread#start":
				emitter.fileImports.add(JRT);
				return "jrt.ThreadStart(" + recv(mi) + ")";
			case "java.lang.Thread#join":
				emitter.fileImports.add(JRT);
				return "jrt.ThreadJoin(" + recv(mi) + ")";
			// Object monitors: the one global jrt monitor (see ControlFlowEmitter.emitSynchronized).
			case "java.lang.Object#wait":
				emitter.fileImports.add(JRT);
				return "jrt.MonitorWait()";
			case "java.lang.Object#notifyAll", "java.lang.Object#notify":
				emitter.fileImports.add(JRT);
				return "jrt.MonitorNotifyAll()";
			// Locale has no rule of its own: jrt.Locale carries the default's language tag.
			case "java.util.Locale#getDefault":
				emitter.fileImports.add(JRT);
				return "jrt.LocaleDefault()";
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

	/** System.out/err.println(x) picks the stream by name; a PrintStream variable is an any holding an io.Writer. */
	private String println(MethodInvocation mi) {
		if (mi.getExpression() == null) return null;
		emitter.fileImports.add("fmt");
		String stream;
		if (mi.getExpression() instanceof QualifiedName qn && qn.getQualifier().getFullyQualifiedName().equals("System")) {
			emitter.fileImports.add("os");
			stream = qn.getName().getIdentifier().equals("err") ? "os.Stderr" : "os.Stdout";
		} else {
			emitter.fileImports.add("io");
			stream = recv(mi) + ".(io.Writer)";
		}
		String a = mi.arguments().isEmpty() ? "" : ", " + arg(mi, 0);
		return "fmt.Fprintln(" + stream + a + ")";
	}
}
