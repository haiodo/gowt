package fixture;

public class Ctors {
	int x;
	String name;
	public Ctors() { this(0); }
	public Ctors(int x) { this(x, "n"); }
	public Ctors(int x, String name) { this.x = x; this.name = name; }
	static Ctors make() { return new Ctors(3); }
	static Ctors make(String n) { return new Ctors(1, n); }
}
