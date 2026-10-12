package fixture;

public class Compound {
	byte b;
	short s;
	char c;
	void run(int k) {
		b += k;
		s *= 2;
		c++;
		b >>= 1;
		s |= 0x10;
		int x = 5;
		x <<= k;
		x ^= -1;
		long l = x;
		l >>>= 3;
		double d = 1.5;
		d += x;
		int y = (int) d;
		y -= (int) l;
	}
}
