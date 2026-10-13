package fixture;

public class Loops1 {
	int sum(int[] a) {
		int t = 0;
		for (int i = 0; i < a.length; i++) t += a[i];
		for (int v : a) t += v;
		int j = 0;
		while (j < 3) { j++; if (j == 2) continue; t += j; }
		do { t--; } while (t > 100);
		return t;
	}
	int labeled() {
		int c = 0;
		outer:
		for (int i = 0; i < 3; i++) {
			for (int k = 0; k < 3; k++) {
				if (k == 2) continue outer;
				if (i == 2) break outer;
				c++;
			}
		}
		return c;
	}
}
