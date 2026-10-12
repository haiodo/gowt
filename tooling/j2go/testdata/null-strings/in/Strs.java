package fixture;

public class Strs {
	String name;
	boolean has() { return name != null && !name.isEmpty(); }
	int len(String s) { return s == null ? 0 : s.length(); }
	String join(String a, String b) { return a + b; }
	String orEmpty(String s) { if (s == null) return ""; return s; }
	String sub(String s) { return s.substring(1, 3); }
	boolean same(String a, String b) { return a.equals(b); }
	void reset() { name = null; }
}
