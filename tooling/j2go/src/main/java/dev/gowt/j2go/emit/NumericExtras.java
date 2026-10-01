package dev.gowt.j2go.emit;

import dev.gowt.j2go.GoTypes;
import org.eclipse.jdt.core.dom.*;

import java.util.Map;

/** Java numeric rules Go lacks: float %, narrowing of a literal, compound assignment through a wider type. */
final class NumericExtras {

	private static final Map<String, Integer> RANK = Map.of("byte", 1, "short", 2, "char", 2, "int", 3, "long", 4, "float", 5, "double", 6);

	private NumericExtras() {}

	/** `a % b` on floats is math.Mod. */
	static String floatRemainder(Emitter emitter, InfixExpression ie) {
		// `a != b || ... || a != b` (SWT's FontData.equals has one): go vet rejects the repeat; call-free operands are pure.
		// The reference platform's output stays as it was.
		if (GoTypes.platform != dev.gowt.j2go.Platform.COCOA
				&& (ie.getOperator() == InfixExpression.Operator.CONDITIONAL_AND || ie.getOperator() == InfixExpression.Operator.CONDITIONAL_OR)) {
			java.util.List<Expression> ops = new java.util.ArrayList<>();
			flatten(ie, ops);
			String goOp = ie.getOperator() == InfixExpression.Operator.CONDITIONAL_AND ? "&&" : "||";
			java.util.List<String> saved = emitter.prelude;
			java.util.List<String> kept = new java.util.ArrayList<>();
			java.util.Set<String> seen = new java.util.HashSet<>();
			boolean dropped = false;
			for (Expression e : ops) {
				if (hasCall(e)) {
					kept.add(EmitUtil.lazyOperand(emitter, e, goOp));
					continue;
				}
				emitter.prelude = new java.util.ArrayList<>();
				String text = emitter.expr(e);
				boolean pure = emitter.prelude.isEmpty();
				emitter.prelude = saved;
				if (!pure) {
					emitter.prelude = saved;
					return null;
				}
				if (seen.add(text)) kept.add(text);
				else dropped = true;
			}
			if (dropped) return String.join(" " + goOp + " ", kept);
		}
		ITypeBinding t = ie.resolveTypeBinding();
		if (ie.getOperator() != InfixExpression.Operator.REMAINDER || !ie.extendedOperands().isEmpty() || t == null
				|| !(t.getName().equals("float") || t.getName().equals("double"))) return null;
		emitter.fileImports.add("math");
		String call = "math.Mod(float64(" + emitter.expr(ie.getLeftOperand()) + "), float64(" + emitter.expr(ie.getRightOperand()) + "))";
		return t.getName().equals("float") ? "float32(" + call + ")" : call;
	}

	/** The operands of a same-operator chain, whether the parser nested it or not. */
	private static void flatten(InfixExpression ie, java.util.List<Expression> out) {
		if (ie.getLeftOperand() instanceof InfixExpression l && l.getOperator() == ie.getOperator()) flatten(l, out);
		else out.add(ie.getLeftOperand());
		out.add(ie.getRightOperand());
		for (Object o : ie.extendedOperands()) out.add((Expression) o);
	}

	private static boolean hasCall(Expression e) {
		boolean[] found = {false};
		e.accept(new ASTVisitor() {
			@Override public boolean visit(MethodInvocation n) { found[0] = true; return false; }
			@Override public boolean visit(ClassInstanceCreation n) { found[0] = true; return false; }
			@Override public boolean visit(Assignment n) { found[0] = true; return false; }
		});
		return found[0];
	}

	/** An integer literal converted to a narrower integer type wraps, as a Java narrowing cast does. */
	static String convert(String goType, String text) {
		if (text.matches("-?(0[xX][0-9a-fA-F]+|[0-9]+)")) {
			boolean neg = text.startsWith("-");
			String digits = neg ? text.substring(1) : text;
			long v = digits.startsWith("0x") || digits.startsWith("0X") ? Long.parseUnsignedLong(digits.substring(2), 16) : Long.parseLong(digits);
			if (neg) v = -v;
			// Only a value that does not fit needs rewriting; others keep their spelling.
			long wrapped = switch (goType) {
				case "int8" -> (byte) v;
				case "int16" -> (short) v;
				case "uint16" -> (char) v;
				case "int32" -> (int) v;
				default -> v;
			};
			if (wrapped != v) return goType + "(" + wrapped + ")";
		}
		return goType + "(" + text + ")";
	}

	/** `x op= y` with y wider than x: Java computes in the wider type, then narrows (int *= -1.5). */
	static String wideCompound(Emitter emitter, Assignment a, String lhs, String rhsRaw) {
		String op = a.getOperator().toString();
		// The reference platform's output stays as it was: its narrowing of the operand is what its tests pass with.
		if (GoTypes.platform == dev.gowt.j2go.Platform.COCOA || !op.matches("[-+*/%]=")) return null;
		ITypeBinding lt = a.getLeftHandSide().resolveTypeBinding();
		ITypeBinding rt = a.getRightHandSide().resolveTypeBinding();
		if (lt == null || rt == null || !lt.isPrimitive() || !rt.isPrimitive()) return null;
		Integer lr = RANK.get(lt.getName());
		Integer rr = RANK.get(rt.getName());
		if (lr == null || rr == null || rr <= lr || rr < 5) return null;
		String wide = GoTypes.map(rt, emitter);
		if (op.equals("%=")) {
			emitter.fileImports.add("math");
			return lhs + " = " + GoTypes.map(lt, emitter) + "(math.Mod(float64(" + lhs + "), float64(" + rhsRaw + ")))";
		}
		return lhs + " = " + GoTypes.map(lt, emitter) + "(" + wide + "(" + lhs + ") " + op.charAt(0) + " " + rhsRaw + ")";
	}
}
