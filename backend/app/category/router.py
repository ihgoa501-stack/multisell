"""分类管理 - 路由"""
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result
from app.common.utils import build_tree
from app.category.schemas import CategoryCreate, CategoryUpdate, CategoryVO
from app.category.service import CategoryService
from app.models import Category

router = APIRouter(tags=["分类管理"])


def category_to_vo(cat: Category, children: list = None) -> CategoryVO:
    return CategoryVO(
        id=cat.id,
        name=cat.name,
        parent_id=cat.parent_id,
        level=cat.level,
        sort_order=cat.sort_order,
        status=cat.status,
        children=children or [],
        created_at=cat.created_at,
        updated_at=cat.updated_at,
    )


@router.post("/categories", summary="创建分类")
async def create_category(data: CategoryCreate, db: AsyncSession = Depends(get_db)):
    cat = await CategoryService.create(db, data.name, data.parent_id, data.sort_order)
    return Result.ok(category_to_vo(cat))


@router.put("/categories/{category_id}", summary="更新分类")
async def update_category(category_id: int, data: CategoryUpdate, db: AsyncSession = Depends(get_db)):
    cat = await CategoryService.update(db, category_id, data.model_dump(exclude_unset=True))
    if not cat:
        return Result.not_found("分类不存在")
    return Result.ok(category_to_vo(cat))


@router.get("/categories/tree", summary="分类树")
async def get_category_tree(db: AsyncSession = Depends(get_db)):
    categories = await CategoryService.get_tree(db)
    # 构建树
    items = []
    for cat in categories:
        items.append({
            "id": cat.id,
            "name": cat.name,
            "parent_id": cat.parent_id,
            "level": cat.level,
            "sort_order": cat.sort_order,
            "status": cat.status,
            "created_at": cat.created_at,
            "updated_at": cat.updated_at,
        })
    tree = build_tree(items, parent_id=0)
    return Result.ok(tree)


@router.delete("/categories/{category_id}", summary="删除分类")
async def delete_category(category_id: int, db: AsyncSession = Depends(get_db)):
    ok, message = await CategoryService.delete(db, category_id)
    if not ok:
        return Result.error(message=message)
    return Result.ok(message=message)
