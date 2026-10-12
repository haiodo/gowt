package fixture;

public class Rec2 {
	record Pair<A, B>(A first, B second) {
		A getFirst() { return first; }
	}
	record Named(String name, int id) {
		String label() { return name + "#" + id; }
	}
	String use() { return new Named("n", 3).label(); }
}
