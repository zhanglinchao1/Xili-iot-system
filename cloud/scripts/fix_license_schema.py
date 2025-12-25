#!/usr/bin/env python3
"""
从config.yaml读取配置并执行数据库迁移脚本
使用psql命令执行，避免需要安装Python数据库驱动
"""
import yaml
import subprocess
import sys
import os
from pathlib import Path

def load_config(config_path=None):
    """从config.yaml加载配置"""
    if config_path is None:
        # 获取脚本所在目录
        script_dir = Path(__file__).parent
        project_root = script_dir.parent
        config_path = project_root / "config.yaml"
    
    with open(config_path, 'r', encoding='utf-8') as f:
        config = yaml.safe_load(f)
    
    return config

def run_migration():
    """执行数据库迁移"""
    try:
        # 加载配置
        config = load_config()
        
        # 获取常规数据库配置
        db_config = config.get('database', {}).get('postgres', {})
        
        # 迁移专用配置（如果存在）
        migration_config = config.get('database', {}).get('migration', {})
        use_sudo = migration_config.get('use_sudo', False) if migration_config else False
        
        if migration_config:
            # 合并迁移配置，迁移配置优先，但保留host、port、dbname等基础配置
            db_config = {
                **db_config,
                'user': migration_config.get('user', db_config.get('user')),
                'password': migration_config.get('password', db_config.get('password'))
            }
        
        # 从环境变量覆盖（如果设置了）
        db_params = {
            'host': os.getenv('DB_HOST', db_config.get('host', 'localhost')),
            'port': os.getenv('DB_PORT', str(db_config.get('port', 5432))),
            'user': os.getenv('DB_USER', db_config.get('user', 'cloud_user')),
            'password': os.getenv('DB_PASSWORD', db_config.get('password', '')),
            'database': os.getenv('DB_NAME', db_config.get('dbname', 'cloud_system'))
        }
        
        migration_file = 'migrations/009_fix_license_schema.sql'
        
        # 获取脚本所在目录
        script_dir = Path(__file__).parent
        project_root = script_dir.parent
        migration_path = project_root / migration_file
        
        if not migration_path.exists():
            print(f"❌ 迁移文件未找到: {migration_path}")
            return 1
        
        print(f"执行数据库迁移: {migration_file}")
        print(f"使用用户: {db_params['user']}")
        print(f"数据库: {db_params['database']}@{db_params['host']}:{db_params['port']}")
        print()
        
        # 设置PGPASSWORD环境变量
        env = os.environ.copy()
        if db_params['password']:
            env['PGPASSWORD'] = db_params['password']
        
        # 使用psql执行迁移
        if use_sudo:
            # 使用sudo执行，不需要数据库密码
            cmd = [
                'sudo', '-u', 'postgres',
                'psql',
                '-h', db_params['host'],
                '-p', db_params['port'],
                '-U', db_params['user'],
                '-d', db_params['database'],
                '-f', str(migration_path)
            ]
            # sudo需要密码，使用echo传递
            sudo_password = os.getenv('SUDO_PASSWORD', '0000')
            result = subprocess.run(
                ['echo', sudo_password],
                stdout=subprocess.PIPE,
                text=True
            )
            sudo_input = result.stdout.encode()
            result = subprocess.run(
                cmd,
                input=sudo_input,
                capture_output=True,
                text=True,
                env=env
            )
        else:
            cmd = [
                'psql',
                '-h', db_params['host'],
                '-p', db_params['port'],
                '-U', db_params['user'],
                '-d', db_params['database'],
                '-f', str(migration_path)
            ]
            result = subprocess.run(cmd, env=env, capture_output=True, text=True)
        
        if result.returncode == 0:
            print("✅ 迁移成功完成！")
            if result.stdout:
                print(result.stdout)
            return 0
        else:
            print("❌ 迁移失败:")
            if result.stderr:
                print(result.stderr)
            if result.stdout:
                print(result.stdout)
            
            # 检查是否是权限错误
            error_output = result.stderr + result.stdout
            if 'must be owner' in error_output or 'permission denied' in error_output.lower():
                print("\n提示: 请使用postgres超级用户执行迁移，或在config.yaml中配置migration用户")
            
            return 1
            
    except FileNotFoundError as e:
        print(f"❌ 配置文件未找到: {e}")
        return 1
    except yaml.YAMLError as e:
        print(f"❌ 配置文件解析失败: {e}")
        return 1
    except Exception as e:
        print(f"❌ 执行失败: {e}")
        import traceback
        traceback.print_exc()
        return 1

if __name__ == '__main__':
    # 切换到项目根目录
    script_dir = Path(__file__).parent
    project_root = script_dir.parent
    os.chdir(project_root)
    
    sys.exit(run_migration())

