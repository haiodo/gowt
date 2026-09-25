package dev.gowt.j2go.emit;

import dev.gowt.j2go.GoTypes;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.*;

/** JUnit 5 -> internal/junit: Assertions calls and a per-class junit.Register(...) registry, with
 * annotations evaluated at translation time for macOS (README "Round 12 tests"). */
final class TestEmitter {

	private static final String JUNIT_IMPORT = "github.com/haiodo/gowt/internal/junit";
	private static final String API = "org.junit.jupiter.api.";
	private static final String COND = "org.junit.jupiter.api.condition.";
	private static final String PARAMS = "org.junit.jupiter.params.";
	// Annotations a test method may carry without being skipped as an unsupported shape.
	private static final Set<String> KNOWN = Set.of(API + "Test", PARAMS + "ParameterizedTest", PARAMS + "provider.ValueSource",
			API + "Tag", API + "Tags", API + "Disabled", COND + "DisabledOnOs", COND + "EnabledOnOs",
			COND + "DisabledIfEnvironmentVariable", COND + "DisabledIfSystemProperty", API + "Timeout", API + "Order",
			API + "DisplayName");

	private final Emitter emitter;

	TestEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	// ---------------------------------------------------------------- Assertions / Assumptions

	/** junit.AssertX(...) for an Assertions/Assumptions call, or null for any other method. */
	String junitCall(MethodInvocation mi, IMethodBinding mb) {
		String q = mb.getDeclaringClass().getErasure().getQualifiedName();
		if (!q.equals(API + "Assertions") && !q.equals(API + "Assumptions")) return null;
		emitter.fileImports.add(JUNIT_IMPORT);
		String name = Names.capitalize(mb.getName());
		ITypeBinding[] pt = mb.getParameterTypes();
		List<?> args = mi.arguments();
		int first = 0;
		String typeArg = "";
		// assertThrows(X.class, ...)/assertInstanceOf(X.class, ...): the class becomes the Go type argument.
		if (pt.length > 0 && pt[0].getErasure().getQualifiedName().equals("java.lang.Class")) {
			typeArg = "[" + GoTypes.map(mb.getReturnType(), emitter) + "]";
			first = 1;
		}
		boolean delta = pt.length >= 3 && pt[2].isPrimitive() && (pt[2].getName().equals("double") || pt[2].getName().equals("float"));
		if (delta && (name.equals("AssertEquals") || name.equals("AssertNotEquals") || name.equals("AssertArrayEquals"))) name += "Delta";
		List<String> out = new ArrayList<>();
		for (int i = first; i < args.size(); i++) {
			Expression a = (Expression) args.get(i);
			String fn = emitter.rawFunc(a);
			boolean varargSlot = mb.isVarargs() && i >= pt.length - 1;
			out.add(fn != null ? fn : varargSlot ? emitter.expr(a) : typedArg(emitter.adaptArg(a, mb, i), pt[i], a.resolveTypeBinding()));
		}
		return "junit." + name + typeArg + "(" + String.join(", ", out) + ")";
	}

	/** The shim takes any: an untyped Go constant would arrive as int/float64, not Java's int32/float32. */
	private String typedArg(String text, ITypeBinding param, ITypeBinding arg) {
		ITypeBinding t = param.isPrimitive() ? param : arg != null && arg.isPrimitive() ? arg : null;
		if (t == null || t.getName().equals("boolean")) return text;
		return GoTypes.map(t, emitter) + "(" + text + ")";
	}

	// ---------------------------------------------------------------- test registry

	/** func init() { junit.Register(...) } for a concrete class with tests, else "". */
	String registration(TypeDeclaration td, TypeModel.ClassInfo ci) {
		ITypeBinding cls = td.resolveBinding();
		if (!emitter.currentGoPackage.equals("swttests") || Modifier.isAbstract(cls.getModifiers()) || cls.isMember()) return "";
		List<IMethodBinding> methods = effectiveMethods(cls);
		List<String> tests = new ArrayList<>();
		List<String> beforeEach = new ArrayList<>(), afterEach = new ArrayList<>(), beforeAll = new ArrayList<>(), afterAll = new ArrayList<>();
		String recv = "t.(*" + ci.goTypeName + ")";
		for (IMethodBinding mb : orderTests(cls, methods)) {
			Set<String> anns = annotationNames(mb.getAnnotations());
			String call = Modifier.isStatic(mb.getModifiers())
					? emitter.staticMethodGoName(mb, emitter.model.lookup(mb.getDeclaringClass())) + "()"
					: emitter.instanceCall(recv, mb, List.of());
			if (anns.contains(API + "BeforeEach")) beforeEach.add("func(t any) { " + call + " }");
			if (anns.contains(API + "AfterEach")) afterEach.add(0, "func(t any) { " + call + " }");
			if (anns.contains(API + "BeforeAll")) beforeAll.add("func() { " + call + " }");
			if (anns.contains(API + "AfterAll")) afterAll.add(0, "func() { " + call + " }");
			if (anns.contains(API + "Test") || anns.contains(PARAMS + "ParameterizedTest")) tests.addAll(testEntries(cls, mb, anns, recv));
		}
		if (tests.isEmpty()) return "";
		emitter.fileImports.add(JUNIT_IMPORT);
		StringBuilder b = new StringBuilder("func init() {\n\tjunit.Register(&junit.Class{\n");
		b.append("\t\tName: \"").append(cls.getName()).append("\",\n");
		b.append("\t\tNew: func() any { return ").append(constructor(cls, ci)).append("() },\n");
		appendList(b, "BeforeAll", "func()", beforeAll);
		appendList(b, "AfterAll", "func()", afterAll);
		appendList(b, "BeforeEach", "func(any)", beforeEach);
		appendList(b, "AfterEach", "func(any)", afterEach);
		b.append("\t\tTests: []junit.Test{\n");
		for (String t : tests) b.append("\t\t\t").append(t).append(",\n");
		b.append("\t\t},\n\t})\n}\n\n");
		return b.toString();
	}

	private static void appendList(StringBuilder b, String field, String type, List<String> items) {
		if (items.isEmpty()) return;
		b.append("\t\t").append(field).append(": []").append(type).append("{\n");
		for (String i : items) b.append("\t\t\t").append(i).append(",\n");
		b.append("\t\t},\n");
	}

	/** One entry per test, a @ParameterizedTest expanded per @ValueSource value. */
	private List<String> testEntries(ITypeBinding cls, IMethodBinding mb, Set<String> anns, String recv) {
		String name = mb.getName();
		StringBuilder common = new StringBuilder();
		List<String> tags = new ArrayList<>(tags(cls.getAnnotations()));
		for (ITypeBinding t = cls.getSuperclass(); t != null; t = t.getSuperclass()) tags.addAll(tags(t.getAnnotations()));
		tags.addAll(tags(mb.getAnnotations()));
		if (!tags.isEmpty()) common.append(", Tags: []string{").append(String.join(", ", tags.stream().map(EmitUtil::goStringLiteral).toList())).append("}");
		long timeout = timeoutNanos(mb.getAnnotations());
		if (timeout > 0) common.append(", Timeout: ").append(timeout);
		String skip = skipReason(cls, mb, anns);
		if (skip != null) return List.of("{Name: " + EmitUtil.goStringLiteral(name) + common + ", Skip: " + skip + "}");
		int params = mb.getParameterTypes().length;
		if (!anns.contains(PARAMS + "ParameterizedTest")) {
			if (params > 0) return List.of(skipped(name, common, "parameter injection not supported"));
			return List.of(entry(name, common, emitter.instanceCall(recv, mb, List.of())));
		}
		IAnnotationBinding values = find(mb.getAnnotations(), PARAMS + "provider.ValueSource");
		if (values == null || params != 1) return List.of(skipped(name, common, "parameter source not supported: " + sources(mb)));
		List<String> out = new ArrayList<>();
		Object[] vs = new Object[0];
		for (IMemberValuePairBinding p : values.getAllMemberValuePairs()) {
			if (asArray(p.getValue()).length > 0) vs = asArray(p.getValue());
		}
		for (int i = 0; i < vs.length; i++) {
			String lit = vs[i] instanceof String s ? EmitUtil.goStringLiteral(s) : String.valueOf(vs[i]);
			out.add(entry(name + "[" + (i + 1) + "]", common, emitter.instanceCall(recv, mb, List.of(lit))));
		}
		return out;
	}

	private static String entry(String name, CharSequence common, String call) {
		return "{Name: " + EmitUtil.goStringLiteral(name) + common + ", Run: func(t any) { " + call + " }}";
	}

	private static String skipped(String name, CharSequence common, String reason) {
		return "{Name: " + EmitUtil.goStringLiteral(name) + common + ", Skip: " + EmitUtil.goStringLiteral(reason) + "}";
	}

	private static String sources(IMethodBinding mb) {
		List<String> s = new ArrayList<>();
		for (IAnnotationBinding a : mb.getAnnotations()) {
			if (a.getAnnotationType().getQualifiedName().startsWith(PARAMS + "provider.")) s.add("@" + a.getName());
		}
		return String.join(" ", s);
	}

	/** Go string expression for why the test is skipped on macOS, or null to run it. */
	private String skipReason(ITypeBinding cls, IMethodBinding mb, Set<String> anns) {
		for (String a : anns) {
			if (a.startsWith("org.junit.") && !KNOWN.contains(a)) return EmitUtil.goStringLiteral("unsupported annotation @" + a);
		}
		List<IAnnotationBinding> all = new ArrayList<>(List.of(mb.getAnnotations()));
		for (ITypeBinding t = cls; t != null; t = t.getSuperclass()) all.addAll(List.of(t.getAnnotations()));
		for (IAnnotationBinding a : all) {
			String reason = condition(a);
			if (reason != null) return reason;
		}
		return null;
	}

	private String condition(IAnnotationBinding a) {
		String q = a.getAnnotationType().getQualifiedName();
		String why = (String) member(a, q.equals(API + "Disabled") ? "value" : "disabledReason");
		String reason = EmitUtil.goStringLiteral("@" + a.getName() + (why == null || why.isEmpty() ? "" : ": " + why.strip()));
		switch (q) {
			case API + "Disabled":
				return reason;
			case COND + "DisabledOnOs", COND + "EnabledOnOs": {
				boolean onMac = false;
				for (Object os : asArray(member(a, "value"))) onMac |= ((IVariableBinding) os).getName().equals("MAC");
				return onMac == q.equals(COND + "DisabledOnOs") ? reason : null;
			}
			case COND + "DisabledIfEnvironmentVariable":
				emitter.fileImports.add(JUNIT_IMPORT);
				return "junit.SkipIfEnv(" + EmitUtil.goStringLiteral((String) member(a, "named")) + ", "
						+ EmitUtil.goStringLiteral((String) member(a, "matches")) + ", " + reason + ")";
			default:
				// DisabledIfSystemProperty: Go has no Java system properties, the property is never set.
				return null;
		}
	}

	private static long timeoutNanos(IAnnotationBinding[] anns) {
		IAnnotationBinding t = find(anns, API + "Timeout");
		if (t == null) return 0;
		long v = ((Number) member(t, "value")).longValue();
		Object unit = member(t, "unit");
		String u = unit instanceof IVariableBinding vb ? vb.getName() : "SECONDS";
		return java.util.concurrent.TimeUnit.valueOf(u).toNanos(v);
	}

	/** Every method the class's tests see: each signature's most-derived declaration, superclass first. */
	private static List<IMethodBinding> effectiveMethods(ITypeBinding cls) {
		List<ITypeBinding> chain = new ArrayList<>();
		for (ITypeBinding t = cls; t != null && !t.getQualifiedName().equals("java.lang.Object"); t = t.getSuperclass()) chain.add(0, t);
		List<IMethodBinding> out = new ArrayList<>();
		for (int i = 0; i < chain.size(); i++) {
			for (IMethodBinding m : chain.get(i).getDeclaredMethods()) {
				if (m.isConstructor() || overriddenBelow(m, chain.subList(i + 1, chain.size()))) continue;
				out.add(m);
			}
		}
		return out;
	}

	private static boolean overriddenBelow(IMethodBinding m, List<ITypeBinding> subclasses) {
		for (ITypeBinding t : subclasses) {
			for (IMethodBinding s : t.getDeclaredMethods()) {
				if (s.overrides(m)) return true;
			}
		}
		return false;
	}

	/** @TestMethodOrder(OrderAnnotation / MethodName) on the class; declaration order otherwise. */
	private static List<IMethodBinding> orderTests(ITypeBinding cls, List<IMethodBinding> methods) {
		IAnnotationBinding order = null;
		for (ITypeBinding t = cls; t != null && order == null; t = t.getSuperclass()) order = find(t.getAnnotations(), API + "TestMethodOrder");
		if (order == null) return methods;
		String orderer = ((ITypeBinding) member(order, "value")).getName();
		List<IMethodBinding> sorted = new ArrayList<>(methods);
		if (orderer.equals("MethodName")) {
			sorted.sort(Comparator.comparing(IMethodBinding::getName));
		} else if (orderer.equals("OrderAnnotation")) {
			sorted.sort(Comparator.comparingInt(m -> {
				IAnnotationBinding o = find(m.getAnnotations(), API + "Order");
				return o == null ? Integer.MAX_VALUE / 2 : ((Number) member(o, "value")).intValue();
			}));
		}
		return sorted;
	}

	private String constructor(ITypeBinding cls, TypeModel.ClassInfo ci) {
		String prefix = (Modifier.isPublic(cls.getModifiers()) ? "New" : "new") + ci.goFuncPrefix;
		for (IMethodBinding m : cls.getDeclaredMethods()) {
			if (m.isConstructor() && m.getParameterTypes().length == 0 && !m.isDefaultConstructor()) {
				return emitter.ctorGoName(m, (Modifier.isPublic(m.getModifiers()) ? "New" : "new") + ci.goFuncPrefix);
			}
		}
		return prefix;
	}

	private static List<String> tags(IAnnotationBinding[] anns) {
		List<String> out = new ArrayList<>();
		for (IAnnotationBinding a : anns) {
			String q = a.getAnnotationType().getQualifiedName();
			if (q.equals(API + "Tag")) out.add((String) member(a, "value"));
			if (q.equals(API + "Tags")) {
				for (Object t : asArray(member(a, "value"))) out.add((String) member((IAnnotationBinding) t, "value"));
			}
		}
		return out;
	}

	private static Set<String> annotationNames(IAnnotationBinding[] anns) {
		Set<String> s = new LinkedHashSet<>();
		for (IAnnotationBinding a : anns) s.add(a.getAnnotationType().getQualifiedName());
		return s;
	}

	private static IAnnotationBinding find(IAnnotationBinding[] anns, String qualified) {
		for (IAnnotationBinding a : anns) {
			if (a.getAnnotationType().getQualifiedName().equals(qualified)) return a;
		}
		return null;
	}

	private static Object member(IAnnotationBinding a, String name) {
		for (IMemberValuePairBinding p : a.getAllMemberValuePairs()) {
			if (p.getName().equals(name)) return p.getValue();
		}
		return null;
	}

	private static Object[] asArray(Object v) {
		return v instanceof Object[] arr ? arr : v == null ? new Object[0] : new Object[]{v};
	}
}
