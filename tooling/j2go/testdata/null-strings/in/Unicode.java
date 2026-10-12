package fixture;

public class Unicode {
	int at(String s, int i) { return s.charAt(i); }
	String up(String s) { return s.toUpperCase(); }
	int idx(String s) { return s.indexOf("x"); }
	String build(int n) {
		StringBuilder sb = new StringBuilder();
		for (int i = 0; i < n; i++) sb.append(i).append(',');
		return sb.toString();
	}
	String[] parts(String s) { return s.split(","); }
}
