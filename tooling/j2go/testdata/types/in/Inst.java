package fixture;

public class Inst {
	static class A {}
	static class B extends A {}
	boolean isB(A a) { return a instanceof B; }
	B asB(A a) { return (B) a; }
	int kind(Object o) {
		if (o instanceof String s) return s.length();
		if (o instanceof Integer) return 1;
		return 0;
	}
}
