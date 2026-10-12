package fixture;

public class Throws {
	static class MyEx extends RuntimeException {
		MyEx(String m) { super(m); }
	}
	void check(int v) {
		if (v < 0) throw new MyEx("neg");
	}
	void rethrow() {
		try {
			check(-1);
		} catch (MyEx e) {
			throw new IllegalStateException(e);
		}
	}
}
