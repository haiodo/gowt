package org.eclipse.swt.tests.junit;

import org.junit.jupiter.api.Disabled;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.condition.EnabledOnOs;
import org.junit.jupiter.api.condition.OS;

@Disabled("class remains disabled")
public class DisabledConditions {
	@Test
	@EnabledOnOs(OS.WINDOWS)
	public void disabledClass() {}
}
