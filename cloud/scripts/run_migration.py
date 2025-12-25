#!/usr/bin/env python3
"""
执行数据库迁移脚本
"""
import psycopg2
import sys
import os

def run_migration():
    # 从环境变量或配置文件读取数据库连接信息
    db_config = {
        'host': os.getenv('DB_HOST', 'localhost'),
        'port': os.getenv('DB_PORT', '5432'),
        'user': os.getenv('DB_USER', 'cloud_user'),
        'password': os.getenv('DB_PASSWORD', 'cloud123456'),
        'database': os.getenv('DB_NAME', 'cloud_system')
    }
    
    migration_file = 'migrations/006_add_cabinet_activation_fields.sql'
    
    try:
        # 连接数据库
        conn = psycopg2.connect(**db_config)
        conn.autocommit = True
        cursor = conn.cursor()
        
        # 读取迁移文件
        with open(migration_file, 'r', encoding='utf-8') as f:
            sql = f.read()
        
        # 执行SQL（忽略权限错误，只关注实际结果）
        print(f"执行迁移: {migration_file}")
        
        # 逐行执行，忽略权限错误
        statements = sql.split(';')
        for stmt in statements:
            stmt = stmt.strip()
            if not stmt or stmt.startswith('--'):
                continue
            try:
                cursor.execute(stmt)
                print(f"✓ 执行成功: {stmt[:50]}...")
            except psycopg2.errors.InsufficientPrivilege:
                print(f"⚠ 权限不足，跳过: {stmt[:50]}...")
            except psycopg2.errors.DuplicateObject:
                print(f"ℹ 对象已存在，跳过: {stmt[:50]}...")
            except Exception as e:
                if 'already exists' in str(e) or 'duplicate' in str(e).lower():
                    print(f"ℹ 已存在，跳过: {stmt[:50]}...")
                else:
                    print(f"✗ 错误: {stmt[:50]}... - {e}")
        
        # 验证字段是否添加成功
        cursor.execute("""
            SELECT column_name 
            FROM information_schema.columns 
            WHERE table_name = 'cabinets' 
            AND column_name IN ('activation_status', 'registration_token', 'api_key')
        """)
        columns = [row[0] for row in cursor.fetchall()]
        
        print(f"\n验证结果:")
        required_columns = ['activation_status', 'registration_token', 'api_key']
        for col in required_columns:
            if col in columns:
                print(f"✓ {col} 字段存在")
            else:
                print(f"✗ {col} 字段缺失")
        
        cursor.close()
        conn.close()
        
        if all(col in columns for col in required_columns):
            print("\n✅ 迁移成功完成！")
            return 0
        else:
            print("\n⚠️  部分字段可能未添加，请检查权限")
            return 1
            
    except Exception as e:
        print(f"❌ 迁移失败: {e}")
        return 1

if __name__ == '__main__':
    # 切换到项目根目录
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(script_dir)
    os.chdir(project_root)
    
    sys.exit(run_migration())

