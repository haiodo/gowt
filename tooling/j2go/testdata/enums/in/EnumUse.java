package fixture;

import java.util.HashMap;
import java.util.Map;

public class EnumUse {
	Level field = Level.MID;
	Map<Level, String> names = new HashMap<>();

	int score(Level l) {
		switch (l) {
			case LOW:
				return 1;
			case MID:
			case HIGH:
				return 3;
			default:
				return 0;
		}
	}

	String arrow(Level l) {
		return switch (l) {
			case LOW -> "l";
			case MID -> "m";
			case HIGH -> "h";
		};
	}

	Level parse(String s) { return Level.valueOf(s); }

	boolean same(Level a, Level b) { return a.equals(b); }

	boolean mixed(Level a, Planet p) { return a.equals(p); }

	boolean viaObject(Level a, Object o, Named n) { return a.equals(o) || a.equals(n); }

	int cmp(Level a, Level b) { return a.compareTo(b); }

	String show(Level l) { return l.name() + l.ordinal() + l; }

	void fill() {
		for (Level l : Level.values()) names.put(l, l.name());
	}

	double gravity() { return Planet.EARTH.surfaceGravity() + Planet.heaviest().ordinal(); }
}
