package fixture;

public class Derived extends Base {
	@Override
	void init() { count++; }
	int count;
	@Override
	public String name() { return "derived:" + super.name(); }
}
