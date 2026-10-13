package fixture;

public class NullObj {
	Object o;
	int[] arr;
	boolean isNull() { return o == null; }
	int len() { return arr == null ? -1 : arr.length; }
	void clear() { o = null; arr = null; }
	Object pick(Object a, Object b) { return a != null ? a : b; }
}
