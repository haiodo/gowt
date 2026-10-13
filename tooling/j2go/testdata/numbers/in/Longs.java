package fixture;

public class Longs {
	static final long MASK = 0xFFFFFFFFL;
	static final int MAX = Integer.MAX_VALUE;
	long hash(String s) {
		long h = 1125899906842597L;
		for (int i = 0; i < s.length(); i++) h = 31 * h + s.charAt(i);
		return h;
	}
	int overflow(int a) { return a * 65536 * 65536; }
	long unsigned(int a) { return a & MASK; }
	int divide(int a, int b) { return a / b % 7; }
	float half(float f) { return f / 2; }
	boolean cmp(long a, int b) { return a < b && b != 0; }
}
