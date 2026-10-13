package fixture;

import java.util.HashMap;
import java.util.Map;

public class Keys {
	record Tag<T>(String name, T value, double w) {}

	Map<Pt, String> names = new HashMap<>();

	boolean same() { return new Pt(1, 2).equals(new Pt(1, 2)); }

	String lookup() {
		names.put(new Pt(1, 2), "a");
		return names.get(new Pt(1, 2)) + new Pt(3, 4) + new Pt(1, 2).hashCode();
	}

	boolean tags(Tag<String> a, Tag<String> b) { return a.equals(b); }
}
