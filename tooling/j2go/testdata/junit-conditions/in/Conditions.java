package org.eclipse.swt.tests.junit;

import org.junit.jupiter.api.Disabled;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.condition.DisabledOnOs;
import org.junit.jupiter.api.condition.EnabledOnOs;
import org.junit.jupiter.api.condition.OS;

@EnabledOnOs(OS.WINDOWS)
public class Conditions {
	@Test
	public void windowsOnly() {}

	@Test
	@DisabledOnOs({OS.MAC, OS.LINUX})
	public void combinedConditions() {}

	@Test
	@Disabled("unconditional")
	public void disabled() {}
}
