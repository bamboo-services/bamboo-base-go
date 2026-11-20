package apperr

// Find 在原因链中查找已经存在的框架统一错误。
//
// 查找是流式的深度优先遍历，判定顺序与 errors.As 一致：
// 取到节点后立刻判定，命中即返回，只有未命中才展开该节点的原因。
// 因此节点实现的标准 As 钩子命中后不会再去访问它的 Unwrap。
//
// 判定内容：先看节点自身类型，再看节点实现的标准 As 钩子，
// 最后才沿 Unwrap 深入子原因。
// 因此通过标准 As 钩子暴露统一错误、但不提供 Unwrap 的包装同样可以被识别。
//
// nil 载体节点与命中 nil 统一错误的投影都算未命中：
// 它们不会遮蔽原因链中其它真实的统一错误。
// 查找过程不设固定深度上限，合法深层包装链会被完整展开，
// 只有真正回到已访问节点的环会被截断。
//
// 本函数只做查找，不构造失败结果，也不修改任何错误；
// 原因链中没有统一错误时返回 nil 指针。
// 错误分类与协议适配可以先调用本函数，避免各自重复实现链遍历。
func Find(cause error) *Error {
	if IsNil(cause) {
		return nil
	}

	visited := make(map[chainIdentity]struct{})
	stack := []error{cause}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if IsNil(current) {
			continue
		}
		if identity, ok := chainIdentityOf(current); ok {
			if _, seen := visited[identity]; seen {
				// 已展开过的节点再次出现即为环，停止展开该分支。
				continue
			}
			visited[identity] = struct{}{}
		}

		// 先判定当前节点：命中即返回，不再访问它的原因。
		if appErr, ok := current.(*Error); ok {
			return appErr
		}
		if appErr, ok := projectStructured(current); ok {
			return appErr
		}

		// 未命中才展开子原因。
		switch wrapper := any(current).(type) {
		case interface{ Unwrap() error }:
			stack = append(stack, wrapper.Unwrap())
		case interface{ Unwrap() []error }:
			children := wrapper.Unwrap()
			for i := len(children) - 1; i >= 0; i-- {
				stack = append(stack, children[i])
			}
		}
	}
	return nil
}
