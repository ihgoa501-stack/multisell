from app.image_gen.router import router
from app.image_gen.canvas_router import router as canvas_router
router.include_router(canvas_router, prefix="")

__all__ = ["router"]
