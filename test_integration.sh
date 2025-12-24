#!/bin/bash

# 多主题功能集成测试脚本

set -e

echo "🚀 开始多主题功能集成测试..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Go环境
check_go_environment() {
    print_status "检查Go环境..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go未安装或不在PATH中"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go版本: $GO_VERSION"
}

# 检查依赖
check_dependencies() {
    print_status "检查测试依赖..."
    
    # 检查testify
    if ! go list -m github.com/stretchr/testify &> /dev/null; then
        print_warning "testify未安装，正在安装..."
        go get github.com/stretchr/testify
    fi
    
    print_success "依赖检查完成"
}

# 运行单元测试
run_unit_tests() {
    print_status "运行单元测试..."
    
    # 运行现有的单元测试
    if go test -v ./... -run "Test.*" -short; then
        print_success "单元测试通过"
    else
        print_error "单元测试失败"
        return 1
    fi
}

# 运行集成测试
run_integration_tests() {
    print_status "运行集成测试..."
    
    # 运行集成测试
    if go test -v -run "TestMultiThemeIntegration" -timeout 30s; then
        print_success "集成测试通过"
    else
        print_error "集成测试失败"
        return 1
    fi
}

# 运行示例应用测试
run_example_tests() {
    print_status "运行示例应用测试..."
    
    cd example-multi-theme
    
    # 检查示例应用的模板文件
    if [ ! -d "templates" ]; then
        print_error "示例应用模板目录不存在"
        cd ..
        return 1
    fi
    
    # 运行示例应用测试
    if go test -v -timeout 60s; then
        print_success "示例应用测试通过"
    else
        print_error "示例应用测试失败"
        cd ..
        return 1
    fi
    
    cd ..
}

# 运行性能测试
run_performance_tests() {
    print_status "运行性能测试..."
    
    # 运行性能基准测试
    if go test -v -run "TestMultiThemePerformance" -timeout 60s; then
        print_success "性能测试通过"
    else
        print_warning "性能测试失败或超时"
    fi
    
    # 运行基准测试
    print_status "运行基准测试..."
    go test -bench=. -benchmem -run=^$ || print_warning "基准测试未找到或失败"
}

# 运行并发测试
run_concurrency_tests() {
    print_status "运行并发安全测试..."
    
    # 运行并发测试
    if go test -v -run "TestMultiThemeConcurrency" -timeout 60s -race; then
        print_success "并发测试通过"
    else
        print_error "并发测试失败"
        return 1
    fi
}

# 验证向后兼容性
verify_backward_compatibility() {
    print_status "验证向后兼容性..."
    
    # 创建临时的传统模式测试
    TEMP_DIR=$(mktemp -d)
    
    # 复制传统模板结构
    mkdir -p "$TEMP_DIR/templates"/{layouts,pages,singles,errors,partials}
    
    # 创建简单的传统模板
    cat > "$TEMP_DIR/templates/layouts/layout.tmpl" << 'EOF'
<!DOCTYPE html>
<html>
<head><title>{{ .title }}</title></head>
<body>{{ template "content" . }}</body>
</html>
EOF

    cat > "$TEMP_DIR/templates/pages/test.tmpl" << 'EOF'
{{ define "content" }}
<h1>{{ .title }}</h1>
<p>传统模式测试</p>
{{ end }}
EOF

    # 创建测试程序
    cat > "$TEMP_DIR/test_legacy.go" << 'EOF'
package main

import (
    "bytes"
    "testing"
    "github.com/nilorg/template"
)

func TestLegacyMode(t *testing.T) {
    engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, nil)
    if err != nil {
        t.Fatal(err)
    }
    defer engine.Close()
    
    if err := engine.Init(); err != nil {
        t.Fatal(err)
    }
    
    var buf bytes.Buffer
    data := template.H{"title": "传统模式"}
    
    if err := engine.RenderPage(&buf, "test", data); err != nil {
        t.Fatal(err)
    }
    
    output := buf.String()
    if !strings.Contains(output, "传统模式") {
        t.Fatal("传统模式渲染失败")
    }
}
EOF

    # 运行传统模式测试
    cd "$TEMP_DIR"
    go mod init legacy-test
    go get github.com/nilorg/template@latest
    
    if go test -v; then
        print_success "向后兼容性验证通过"
    else
        print_error "向后兼容性验证失败"
        cd - > /dev/null
        rm -rf "$TEMP_DIR"
        return 1
    fi
    
    cd - > /dev/null
    rm -rf "$TEMP_DIR"
}

# 生成测试报告
generate_test_report() {
    print_status "生成测试报告..."
    
    REPORT_FILE="test_report_$(date +%Y%m%d_%H%M%S).txt"
    
    {
        echo "多主题功能集成测试报告"
        echo "========================="
        echo "测试时间: $(date)"
        echo "Go版本: $(go version)"
        echo ""
        
        echo "测试覆盖率:"
        go test -coverprofile=coverage.out ./... 2>/dev/null || echo "覆盖率测试失败"
        if [ -f coverage.out ]; then
            go tool cover -func=coverage.out | tail -1
            rm coverage.out
        fi
        echo ""
        
        echo "依赖信息:"
        go list -m all | head -10
        echo ""
        
        echo "测试环境:"
        echo "操作系统: $(uname -s)"
        echo "架构: $(uname -m)"
        echo "CPU核心数: $(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 'unknown')"
        echo "内存: $(free -h 2>/dev/null | grep Mem | awk '{print $2}' || echo 'unknown')"
        
    } > "$REPORT_FILE"
    
    print_success "测试报告已生成: $REPORT_FILE"
}

# 清理函数
cleanup() {
    print_status "清理测试环境..."
    
    # 清理可能的临时文件
    rm -f coverage.out
    rm -f *.test
    
    print_success "清理完成"
}

# 主测试流程
main() {
    echo "========================================"
    echo "    多主题功能集成测试套件"
    echo "========================================"
    echo ""
    
    # 设置错误处理
    trap cleanup EXIT
    
    # 检查环境
    check_go_environment
    check_dependencies
    
    echo ""
    echo "开始执行测试..."
    echo ""
    
    # 运行测试套件
    FAILED_TESTS=()
    
    if ! run_unit_tests; then
        FAILED_TESTS+=("单元测试")
    fi
    
    if ! run_integration_tests; then
        FAILED_TESTS+=("集成测试")
    fi
    
    if ! run_example_tests; then
        FAILED_TESTS+=("示例应用测试")
    fi
    
    if ! run_concurrency_tests; then
        FAILED_TESTS+=("并发测试")
    fi
    
    if ! verify_backward_compatibility; then
        FAILED_TESTS+=("向后兼容性测试")
    fi
    
    # 性能测试不影响整体结果
    run_performance_tests
    
    # 生成报告
    generate_test_report
    
    echo ""
    echo "========================================"
    echo "           测试结果汇总"
    echo "========================================"
    
    if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
        print_success "🎉 所有测试通过！"
        echo ""
        print_success "多主题功能已准备就绪，可以安全部署。"
        exit 0
    else
        print_error "❌ 以下测试失败:"
        for test in "${FAILED_TESTS[@]}"; do
            echo "  - $test"
        done
        echo ""
        print_error "请修复失败的测试后重新运行。"
        exit 1
    fi
}

# 检查命令行参数
case "${1:-}" in
    "unit")
        check_go_environment
        check_dependencies
        run_unit_tests
        ;;
    "integration")
        check_go_environment
        check_dependencies
        run_integration_tests
        ;;
    "example")
        check_go_environment
        check_dependencies
        run_example_tests
        ;;
    "performance")
        check_go_environment
        check_dependencies
        run_performance_tests
        ;;
    "concurrency")
        check_go_environment
        check_dependencies
        run_concurrency_tests
        ;;
    "compatibility")
        check_go_environment
        check_dependencies
        verify_backward_compatibility
        ;;
    "help"|"-h"|"--help")
        echo "用法: $0 [选项]"
        echo ""
        echo "选项:"
        echo "  unit          只运行单元测试"
        echo "  integration   只运行集成测试"
        echo "  example       只运行示例应用测试"
        echo "  performance   只运行性能测试"
        echo "  concurrency   只运行并发测试"
        echo "  compatibility 只运行向后兼容性测试"
        echo "  help          显示此帮助信息"
        echo ""
        echo "不带参数运行将执行完整的测试套件。"
        ;;
    *)
        main
        ;;
esac