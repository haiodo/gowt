package fixture;

public class Leaf extends Derived {
	@Override
	public int size() { return super.size() + 1; }
	void poke() { open(); init(); }
}
