# 商品管理系统 - Alembic 数据库迁移

## 初始化迁移
```bash
cd backend
alembic init alembic
```

## 生成新的迁移
```bash
alembic revision --autogenerate -m "描述变更内容"
```

## 执行迁移
```bash
alembic upgrade head
```

## 回滚
```bash
alembic downgrade -1
```

## 查看状态
```bash
alembic current
alembic history
```
