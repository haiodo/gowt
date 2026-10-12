package fixture;

public class Switch1 {
	String name(int k) {
		switch (k) {
			case 1: return "one";
			case 2:
			case 3: return "few";
			default: return "many";
		}
	}
	int fall(int k) {
		int r = 0;
		switch (k) {
			case 1: r += 1;
			case 2: r += 2; break;
			case 3: r += 3; break;
		}
		return r;
	}
}
