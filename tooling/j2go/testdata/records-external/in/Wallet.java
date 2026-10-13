package fixture;

public class Wallet {
	record Entry(String label, Money amount) {}

	Money total(Entry a, Entry b) { return a.amount().plus(b.amount()); }

	boolean sameLabel(Entry a, Entry b) { return a.label().equals(b.label()) && a.equals(b); }
}
