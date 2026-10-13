package fixture;

public class Ints {
	int add(int a, int b) { return a + b; }
	long mix(int a, long b, short c, byte d) { return a * b + c - d; }
	int shifts(int x) { return (x << 3) | (x >>> 2) ^ (x >> 1); }
	char next(char c) { return (char) (c + 1); }
	int narrow(long v) { return (int) v; }
	double ratio(int a, int b) { return (double) a / b; }
	int counter() {
		int n = 0;
		n += 2;
		n++;
		return n;
	}
}
