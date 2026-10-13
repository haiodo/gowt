package fixture;

@Marker("x")
public class Marked {
	@Marker(value = "f", ids = {1, 2})
	int f;
	Marker last;

	@Marker
	String describe(Marker m, Level l) {
		last = m;
		return m == null ? "none" : "some" + l;
	}
}
