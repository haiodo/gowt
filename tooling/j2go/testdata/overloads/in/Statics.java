package fixture;

public class Statics {
	static int abs(int v) { return v < 0 ? -v : v; }
	static long abs(long v) { return v < 0 ? -v : v; }
	static double abs(double v) { return v < 0 ? -v : v; }
	static int test() {
		return abs(-1) + (int) abs(-2L) + (int) abs(-1.5);
	}
}
