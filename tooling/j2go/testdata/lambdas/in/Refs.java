package fixture;

import java.util.function.Function;
import java.util.function.Supplier;

public class Refs {
	static int dbl(int x) { return x * 2; }
	String tag() { return "t"; }
	Function<Integer, Integer> f() { return Refs::dbl; }
	Supplier<String> s() { return this::tag; }
	Supplier<Refs> ctor() { return Refs::new; }
}
