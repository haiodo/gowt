/*
 * Copyright (c) 2026 Fixture Authors. All rights reserved.
 * Licence header: every generated part file must carry it.
 */
package fixture;

public class Parts {
	static int counter = computeCounter();
	static final String LABEL = "parts";

	static {
		counter += 1;
	}

	int value;
	String name;

	public Parts(int value) {
		this.value = value;
	}

	public Parts(String name) {
		this.name = name;
	}

	static int computeCounter() {
		return 41;
	}

	public int getValue() {
		return value;
	}

	public void setValue(int v) {
		this.value = v;
	}

	public void setValue(String s) {
		this.value = s.length();
	}

	public double scale(double f) {
		return Math.floor(f * counter);
	}

	public String describe() {
		Runnable r = new Runnable() {
			public void run() {
				counter++;
			}
		};
		r.run();
		return LABEL;
	}

	public int useOp(final int k) {
		Op op = new Op() {
			int apply(int x) {
				return x + k;
			}
		};
		return op.twice(2);
	}

	static abstract class Op {
		abstract int apply(int x);

		int twice(int x) {
			return apply(apply(x));
		}
	}

	static class Inner {
		public int doubleIt(int x) {
			return x * 2;
		}
	}

	public int useInner(int x) {
		return new Inner().doubleIt(x);
	}
}
