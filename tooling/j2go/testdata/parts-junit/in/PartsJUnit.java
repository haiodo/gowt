package org.eclipse.swt.tests.junit;

import org.junit.jupiter.api.Test;

public class PartsJUnit {
	@Test
	public void firstCase() {
		helper(1);
	}

	@Test
	public void secondCase() {
		helper(2);
	}

	@Test
	public void thirdCase() {
		helper(3);
	}

	void helper(int n) {
	}
}
