package fixture;

public enum Planet implements Named {
	MERCURY(3.303e+23, 2.4397e6),
	EARTH(5.976e+24, 6.37814e6);

	static final double G = 6.67300E-11;
	static int created = 1;

	private final double mass;
	private final double radius;
	public final String name = "planet";

	Planet(double mass, double radius) {
		this.mass = mass;
		this.radius = radius;
	}

	double surfaceGravity() { return G * mass / (radius * radius); }

	public String label() { return name().toLowerCase(); }

	static Planet heaviest() {
		Planet best = null;
		for (Planet p : values()) {
			if (best == null || p.mass > best.mass) best = p;
		}
		return best;
	}
}
