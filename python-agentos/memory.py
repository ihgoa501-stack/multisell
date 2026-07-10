import sqlite3
import sqlite_vec
import struct
from typing import List, Dict, Optional, Any

class CognitiveMemory:
    def __init__(self, db_path: str = "agentos_memory.db", embedding_dim: int = 1536):
        self.db_path = db_path
        self.embedding_dim = embedding_dim
        self.conn = sqlite3.connect(db_path, check_same_thread=False)
        self.conn.enable_load_extension(True)
        sqlite_vec.load(self.conn)
        self.conn.enable_load_extension(False)
        self._init_db()

    def _init_db(self):
        self.conn.execute("""
            CREATE TABLE IF NOT EXISTS working_memory (
                key TEXT PRIMARY KEY,
                value TEXT,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )
        """)
        self.conn.execute("""
            CREATE TABLE IF NOT EXISTS episodic_metadata (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                context TEXT,
                action TEXT,
                result TEXT,
                timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
            )
        """)
        cursor = self.conn.execute("SELECT name FROM sqlite_master WHERE type='table' AND name='vec_episodic'")
        if not cursor.fetchone():
            self.conn.execute(f"""
                CREATE VIRTUAL TABLE vec_episodic USING vec0(
                    rowid INTEGER PRIMARY KEY,
                    embedding float[{self.embedding_dim}]
                )
            """)
        self.conn.execute("""
            CREATE TABLE IF NOT EXISTS semantic_metadata (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                concept TEXT,
                description TEXT,
                timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
            )
        """)
        cursor = self.conn.execute("SELECT name FROM sqlite_master WHERE type='table' AND name='vec_semantic'")
        if not cursor.fetchone():
            self.conn.execute(f"""
                CREATE VIRTUAL TABLE vec_semantic USING vec0(
                    rowid INTEGER PRIMARY KEY,
                    embedding float[{self.embedding_dim}]
                )
            """)
        self.conn.commit()

    def _pack_vector(self, vector: List[float]) -> bytes:
        if len(vector) != self.embedding_dim:
            raise ValueError(f"Vector dimension mismatch. Expected {self.embedding_dim}, got {len(vector)}")
        return struct.pack(f"<{self.embedding_dim}f", *vector)

    def save_episodic(self, context: str, action: str, result: str, embedding: List[float]) -> int:
        cursor = self.conn.cursor()
        cursor.execute(
            "INSERT INTO episodic_metadata (context, action, result) VALUES (?, ?, ?)",
            (context, action, result)
        )
        rowid = cursor.lastrowid
        packed = self._pack_vector(embedding)
        cursor.execute(
            "INSERT INTO vec_episodic (rowid, embedding) VALUES (?, ?)",
            (rowid, packed)
        )
        self.conn.commit()
        return rowid

    def query_episodic(self, embedding: List[float], limit: int = 5) -> List[Dict[str, Any]]:
        packed = self._pack_vector(embedding)
        cursor = self.conn.cursor()
        cursor.execute(f"""
            SELECT
                m.id,
                m.context,
                m.action,
                m.result,
                m.timestamp,
                v.distance
            FROM vec_episodic v
            JOIN episodic_metadata m ON v.rowid = m.id
            WHERE v.embedding MATCH ? AND k = ?
            ORDER BY v.distance ASC
        """, (packed, limit))
        rows = cursor.fetchall()
        return [
            {
                "id": row[0],
                "context": row[1],
                "action": row[2],
                "result": row[3],
                "timestamp": row[4],
                "distance": row[5]
            }
            for row in rows
        ]

    def save_semantic(self, concept: str, description: str, embedding: List[float]) -> int:
        cursor = self.conn.cursor()
        cursor.execute(
            "INSERT INTO semantic_metadata (concept, description) VALUES (?, ?)",
            (concept, description)
        )
        rowid = cursor.lastrowid
        packed = self._pack_vector(embedding)
        cursor.execute(
            "INSERT INTO vec_semantic (rowid, embedding) VALUES (?, ?)",
            (rowid, packed)
        )
        self.conn.commit()
        return rowid

    def query_semantic(self, embedding: List[float], limit: int = 5) -> List[Dict[str, Any]]:
        packed = self._pack_vector(embedding)
        cursor = self.conn.cursor()
        cursor.execute(f"""
            SELECT
                m.id,
                m.concept,
                m.description,
                m.timestamp,
                v.distance
            FROM vec_semantic v
            JOIN semantic_metadata m ON v.rowid = m.id
            WHERE v.embedding MATCH ? AND k = ?
            ORDER BY v.distance ASC
        """, (packed, limit))
        rows = cursor.fetchall()
        return [
            {
                "id": row[0],
                "concept": row[1],
                "description": row[2],
                "timestamp": row[3],
                "distance": row[4]
            }
            for row in rows
        ]

    def set_working(self, key: str, value: str):
        self.conn.execute(
            "INSERT OR REPLACE INTO working_memory (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
            (key, value)
        )
        self.conn.commit()

    def get_working(self, key: str) -> Optional[str]:
        cursor = self.conn.execute("SELECT value FROM working_memory WHERE key = ?", (key,))
        row = cursor.fetchone()
        return row[0] if row else None

    def delete_working(self, key: str):
        self.conn.execute("DELETE FROM working_memory WHERE key = ?", (key,))
        self.conn.commit()

    def close(self):
        self.conn.close()
