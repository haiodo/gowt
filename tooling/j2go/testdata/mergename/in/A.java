package fixture.a;

import fixture.b.B;

public class A {
	public B peer() {
		return new B(this);
	}
}
