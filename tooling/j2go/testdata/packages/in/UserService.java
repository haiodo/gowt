package fixture.service;

import fixture.model.User;
import java.util.ArrayList;
import java.util.List;

public class UserService {
	static final List<String> SEEN = new ArrayList<>();
	static final User ROOT = new User(String.valueOf(SEEN.size()));

	public User create(String name) {
		SEEN.add(name);
		return new User(name);
	}
}
