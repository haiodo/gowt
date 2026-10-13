package fixture;

public class Dollar2 {
	int a$b = 1;
	int a_b = 2;
	int $ = 3;
	public int $pub = 4;
	public int pub$ = 5;
	int $priv = 6;
	public static final int $MAX = 7;

	public void $run() {}

	public int pub$() { return pub$ + $pub + $priv + $ + a$b + a_b + $MAX; }

	public class $Inner {}
}
