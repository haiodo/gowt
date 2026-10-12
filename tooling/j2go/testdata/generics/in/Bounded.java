package fixture;

public class Bounded<T extends Comparable<T>> {
	T max(T a, T b) { return a.compareTo(b) >= 0 ? a : b; }
	static <K extends CharSequence> int len(K k) { return k.length(); }
}
