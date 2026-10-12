package fixture;

import java.util.function.Function;
import java.util.function.Supplier;

public class Lam1 {
	int base = 3;
	Function<Integer, Integer> adder() { return x -> x + base; }
	Supplier<String> hello() { return () -> "hi"; }
	Runnable r() { return () -> base++; }
	int apply(Function<Integer, Integer> f) { return f.apply(2); }
	int go() { return apply(x -> x * 2); }
}
