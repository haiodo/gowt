package fixture;

public class Anon2 {
	static abstract class Op {
		abstract int apply(int x);
		int twice(int x) { return apply(apply(x)); }
	}
	int use(final int k) {
		Op op = new Op() {
			int apply(int x) { return x + k; }
		};
		return op.twice(1);
	}
}
