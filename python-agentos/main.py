import os
import sys
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional, Dict, Any

# Ensure current directory is in sys.path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from memory import CognitiveMemory

# Define global memory manager
db_path = os.getenv("COGNITIVE_DB_PATH", "agentos_memory.db")
embedding_dim = int(os.getenv("COGNITIVE_EMBEDDING_DIM", "1536"))

# Use a default dim of 1536 but detect from request if possible or keep standard
memory = CognitiveMemory(db_path=db_path, embedding_dim=embedding_dim)

app = FastAPI(title="LingMirror Cognitive Brain Microservice")

class DecideRequest(BaseModel):
    context: str
    embedding: List[float]
    limit: Optional[int] = 5

class EpisodeMatch(BaseModel):
    id: int
    context: str
    action: str
    result: str
    timestamp: str
    distance: float

class DecideResponse(BaseModel):
    decision: str
    similar_episodes: List[EpisodeMatch]

class ReflectRequest(BaseModel):
    context: str
    action: str
    result: str
    embedding: List[float]

class ReflectResponse(BaseModel):
    status: str
    id: int

class WorkingMemorySetRequest(BaseModel):
    key: str
    value: str

class WorkingMemoryGetResponse(BaseModel):
    key: str
    value: Optional[str]

@app.post("/api/v1/brain/decide", response_model=DecideResponse)
async def decide(req: DecideRequest):
    global memory
    try:
        # If dimensions differ, dynamically handle or return error
        if len(req.embedding) != memory.embedding_dim:
            memory.close()
            memory = CognitiveMemory(db_path=db_path, embedding_dim=len(req.embedding))

        episodes = memory.query_episodic(req.embedding, limit=req.limit)

        # Simple heuristic decision logic
        if episodes and episodes[0]["distance"] < 0.1:
            best_match = episodes[0]
            decision = f"Recommending action '{best_match['action']}' based on historical match #{best_match['id']} (distance {best_match['distance']:.4f}) with result '{best_match['result']}'."
        else:
            decision = "No matching historical episode with high confidence. Defaulting to standard workflow."

        similar = [
            EpisodeMatch(
                id=ep["id"],
                context=ep["context"],
                action=ep["action"],
                result=ep["result"],
                timestamp=ep["timestamp"],
                distance=ep["distance"]
            )
            for ep in episodes
        ]
        return DecideResponse(decision=decision, similar_episodes=similar)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/api/v1/brain/reflect", response_model=ReflectResponse)
async def reflect(req: ReflectRequest):
    global memory
    try:
        if len(req.embedding) != memory.embedding_dim:
            memory.close()
            memory = CognitiveMemory(db_path=db_path, embedding_dim=len(req.embedding))

        rowid = memory.save_episodic(
            context=req.context,
            action=req.action,
            result=req.result,
            embedding=req.embedding
        )
        return ReflectResponse(status="success", id=rowid)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/api/v1/brain/working")
async def set_working_mem(req: WorkingMemorySetRequest):
    try:
        memory.set_working(req.key, req.value)
        return {"status": "success"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/api/v1/brain/working/{key}", response_model=WorkingMemoryGetResponse)
async def get_working_mem(key: str):
    try:
        val = memory.get_working(key)
        return WorkingMemoryGetResponse(key=key, value=val)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.delete("/api/v1/brain/working/{key}")
async def delete_working_mem(key: str):
    try:
        memory.delete_working(key)
        return {"status": "success"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.on_event("shutdown")
def shutdown_event():
    memory.close()

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
