package fixture.b;

import fixture.a.A;

public class B {
	private final A owner;

	public B(A owner) {
		this.owner = owner;
	}

	public A getOwner() {
		return owner;
	}
}
