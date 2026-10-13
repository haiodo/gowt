package fixture;

public class Finally {
	int n;
	int run() {
		try {
			n++;
			return n;
		} finally {
			n--;
		}
	}
	void both() {
		try {
			throw new IllegalArgumentException("bad");
		} catch (IllegalArgumentException e) {
			n = 1;
		} finally {
			n += 10;
		}
	}
}
