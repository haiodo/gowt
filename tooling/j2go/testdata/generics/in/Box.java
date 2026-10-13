package fixture;

public class Box<T> {
	T value;
	Box(T v) { value = v; }
	T get() { return value; }
	<R> Box<R> map(java.util.function.Function<T, R> f) { return new Box<R>(f.apply(value)); }
	static <E> E first(java.util.List<E> l) { return l.get(0); }
}
