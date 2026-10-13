package fixture;

public class Nested {
	int value = 4;
	class Inner {
		int get() { return value; }
	}
	static class Holder {
		int v;
		Holder(int v) { this.v = v; }
	}
	int use() {
		Inner in = new Inner();
		Holder h = new Holder(2);
		return in.get() + h.v;
	}
}
