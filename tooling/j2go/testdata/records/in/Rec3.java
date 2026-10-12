package fixture;

public class Rec3 {
	static final class Range {
		final int lo, hi;
		Range(int lo, int hi) { this.lo = lo; this.hi = hi; }
		boolean contains(int v) { return v >= lo && v < hi; }
	}
	boolean in(int v) { return new Range(0, 10).contains(v); }
}
