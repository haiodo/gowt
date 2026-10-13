package fixture;

public enum Op {
	PLUS("+") {
		int apply(int a, int b) { return a + b; }
	},
	TIMES("*") {
		int apply(int a, int b) { return a * b; }

		@Override
		public String toString() { return "times " + sym(); }
	},
	NEG("-") {
		int apply(int a, int b) { return -a; }

		void log(String m) { System.out.println(m + sym()); }
	};

	private final String sym;

	Op(String sym) { this.sym = sym; }

	abstract int apply(int a, int b);

	void log(String m) {}

	String sym() { return sym; }

	static int fold(Op op, int[] xs) {
		int acc = xs[0];
		for (int i = 1; i < xs.length; i++) acc = op.apply(acc, xs[i]);
		return acc;
	}
}
