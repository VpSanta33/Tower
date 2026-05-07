#!/bin/sh

# JWT密钥持久化文件路径（挂载到volume）
JWT_SECRET_FILE="/app/data/.tower_jwt_secret"

# 生成或读取JWT密钥
if [ -n "$TOWER_JWT_SECRET" ]; then
    # 用户通过环境变量指定了密钥
    echo "[INFO] Using TOWER_JWT_SECRET from environment variable"
elif [ -f "$JWT_SECRET_FILE" ]; then
    # 从持久化文件读取
    TOWER_JWT_SECRET=$(cat "$JWT_SECRET_FILE")
    echo "[INFO] Loaded TOWER_JWT_SECRET from persistent storage"
else
    # 首次启动，生成随机密钥并持久化
    mkdir -p /app/data
    TOWER_JWT_SECRET=$(head -c 48 /dev/urandom | base64 | tr -d '\n/+=' | head -c 64)
    echo -n "$TOWER_JWT_SECRET" > "$JWT_SECRET_FILE"
    chmod 600 "$JWT_SECRET_FILE"
    echo "[INFO] Generated new TOWER_JWT_SECRET and saved to persistent storage"
fi

export TOWER_JWT_SECRET

# 使用envsubst替换配置文件中的环境变量
envsubst '${TOWER_JWT_SECRET}' < /app/etc/tower-api.yaml.template > /app/etc/tower-api.yaml

# 执行传入的命令
exec "$@"
