package fixture;

public record Money(long cents, String currency) {
	public Money {
		if (cents < 0) throw new IllegalArgumentException("negative");
	}

	public Money plus(Money o) { return new Money(cents + o.cents(), currency); }

	public String show() { return cents + " " + currency; }
}
