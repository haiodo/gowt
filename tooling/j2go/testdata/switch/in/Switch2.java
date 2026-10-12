package fixture;

public class Switch2 {
	String s(String v) {
		switch (v) {
			case "a": return "A";
			case "b": return "B";
			default: return "?";
		}
	}
	int arrow(int k) {
		return switch (k) {
			case 1, 2 -> 10;
			case 3 -> { int t = k * 2; yield t; }
			default -> 0;
		};
	}
}
