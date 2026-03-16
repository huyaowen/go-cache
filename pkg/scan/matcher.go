package scan

import (
	"strings"
)

// Matcher 接口与实现匹配器
type Matcher struct{}

// NewMatcher 创建匹配器
func NewMatcher() *Matcher {
	return &Matcher{}
}

// Match 匹配接口与实现
// 返回：接口名 -> 匹配结果
func (m *Matcher) Match(interfaces map[string]*InterfaceInfo, services map[string]*ServiceInfo) map[string]*MatchResult {
	results := make(map[string]*MatchResult)

	for ifaceName, iface := range interfaces {
		expectedImpl := m.expectedImplName(ifaceName)

		svc, exists := services[expectedImpl]
		if !exists {
			continue
		}

		if !m.verifyMethods(iface, svc) {
			continue
		}

		svc.Implements = ifaceName
		results[ifaceName] = &MatchResult{
			Interface: iface,
			Service:   svc,
		}
	}

	return results
}

// expectedImplName 根据接口名计算期望的实现名
// 规则：
// - UserServiceInterface → userService
// - OrderService → orderService
// - UserRepository → userRepository
func (m *Matcher) expectedImplName(ifaceName string) string {
	name := strings.TrimSuffix(ifaceName, "Interface")
	if len(name) > 0 {
		return strings.ToLower(name[:1]) + name[1:]
	}
	return strings.ToLower(ifaceName[:1]) + ifaceName[1:]
}

// verifyMethods 验证服务是否实现了接口的所有方法
func (m *Matcher) verifyMethods(iface *InterfaceInfo, svc *ServiceInfo) bool {
	for _, ifaceMethod := range iface.Methods {
		svcMethod, exists := svc.Methods[ifaceMethod.Name]
		if !exists {
			return false
		}

		if !m.matchMethodSignature(ifaceMethod, svcMethod) {
			return false
		}
	}

	return true
}

// matchMethodSignature 匹配方法签名
func (m *Matcher) matchMethodSignature(ifaceMethod *MethodSpec, svcMethod *MethodInfo) bool {
	if len(ifaceMethod.Params) != len(svcMethod.Params) {
		return false
	}

	if len(ifaceMethod.Returns) != len(svcMethod.Returns) {
		return false
	}

	for i, param := range ifaceMethod.Params {
		if !m.matchType(param.Type, svcMethod.Params[i].Type) {
			return false
		}
	}

	for i, ret := range ifaceMethod.Returns {
		if !m.matchType(ret.Type, svcMethod.Returns[i].Type) {
			return false
		}
	}

	return true
}

// matchType 匹配类型
func (m *Matcher) matchType(ifaceType, svcType string) bool {
	if ifaceType == svcType {
		return true
	}

	if strings.TrimPrefix(ifaceType, "*") == strings.TrimPrefix(svcType, "*") {
		return true
	}

	if strings.Contains(ifaceType, ".") && strings.Contains(svcType, ".") {
		ifaceParts := strings.Split(ifaceType, ".")
		svcParts := strings.Split(svcType, ".")
		if ifaceParts[len(ifaceParts)-1] == svcParts[len(svcParts)-1] {
			return true
		}
	}

	return false
}

// MatchAllFiles 匹配文件中所有接口与实现
func (m *Matcher) MatchAllFiles(fileInfo *FileInfo) map[string]*MatchResult {
	return m.Match(fileInfo.Interfaces, fileInfo.Services)
}

// GetServiceByInterface 根据接口名获取服务实现名
func (m *Matcher) GetServiceByInterface(ifaceName string) string {
	return m.expectedImplName(ifaceName)
}

// GetInterfaceCandidates 获取可能的接口候选
func (m *Matcher) GetInterfaceCandidates(svcName string, interfaces map[string]*InterfaceInfo) []string {
	var candidates []string

	for ifaceName := range interfaces {
		if m.expectedImplName(ifaceName) == svcName {
			candidates = append(candidates, ifaceName)
		}
	}

	return candidates
}
