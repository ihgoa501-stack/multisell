import os
import sys
import unittest

# Ensure current directory is in sys.path first so python can find local modules
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

# Set env vars BEFORE importing main.app so global initialization uses test values
os.environ["COGNITIVE_DB_PATH"] = ":memory:"
os.environ["COGNITIVE_EMBEDDING_DIM"] = "4"

from fastapi.testclient import TestClient
from memory import CognitiveMemory
from main import app

class TestCognitiveMemory(unittest.TestCase):
    def setUp(self):
        # Use an in-memory database for testing
        self.db_path = ":memory:"
        self.dim = 4
        self.mem = CognitiveMemory(db_path=self.db_path, embedding_dim=self.dim)

    def tearDown(self):
        self.mem.close()

    def test_working_memory(self):
        # Set
        self.mem.set_working("agent_state", "active")
        # Get
        val = self.mem.get_working("agent_state")
        self.assertEqual(val, "active")
        # Delete
        self.mem.delete_working("agent_state")
        self.assertIsNone(self.mem.get_working("agent_state"))

    def test_episodic_memory(self):
        context = "User asks to list products on Shopee"
        action = "Call shopee publish action with product data"
        result = "Success: product published under ID shp-123"
        embedding = [0.1, 0.2, 0.3, 0.4]

        # Save
        rowid = self.mem.save_episodic(context, action, result, embedding)
        self.assertTrue(rowid > 0)

        # Query exact match (distance should be close to 0)
        results = self.mem.query_episodic(embedding, limit=1)
        self.assertEqual(len(results), 1)
        self.assertEqual(results[0]["id"], rowid)
        self.assertEqual(results[0]["context"], context)
        self.assertEqual(results[0]["action"], action)
        self.assertEqual(results[0]["result"], result)
        self.assertLess(results[0]["distance"], 1e-5)

        # Query with slightly different embedding
        query_emb = [0.11, 0.19, 0.31, 0.39]
        results_approx = self.mem.query_episodic(query_emb, limit=1)
        self.assertEqual(len(results_approx), 1)
        self.assertEqual(results_approx[0]["id"], rowid)
        # Approximate match should have slightly higher distance but still close
        self.assertTrue(0.0 < results_approx[0]["distance"] < 0.1)

    def test_semantic_memory(self):
        concept = "Shopee Publish Rate Limit"
        description = "Shopee allows a maximum of 10 list requests per minute per shop"
        embedding = [0.9, 0.8, 0.7, 0.6]

        rowid = self.mem.save_semantic(concept, description, embedding)
        self.assertTrue(rowid > 0)

        results = self.mem.query_semantic(embedding, limit=1)
        self.assertEqual(len(results), 1)
        self.assertEqual(results[0]["id"], rowid)
        self.assertEqual(results[0]["concept"], concept)
        self.assertEqual(results[0]["description"], description)
        self.assertLess(results[0]["distance"], 1e-5)

class TestCognitiveAPI(unittest.TestCase):
    def setUp(self):
        self.client = TestClient(app)

    def test_api_workflow(self):
        # 1. Save an episode via reflect
        reflect_data = {
            "context": "Inventory alert: stock < 5",
            "action": "Trigger inventory restock request",
            "result": "Restock scheduled successfully",
            "embedding": [0.5, 0.5, 0.5, 0.5]
        }
        res_reflect = self.client.post("/api/v1/brain/reflect", json=reflect_data)
        self.assertEqual(res_reflect.status_code, 200)
        self.assertEqual(res_reflect.json()["status"], "success")
        inserted_id = res_reflect.json()["id"]

        # 2. Query decision based on exact matching context embedding
        decide_data = {
            "context": "Inventory alert: stock < 5",
            "embedding": [0.5, 0.5, 0.5, 0.5],
            "limit": 1
        }
        res_decide = self.client.post("/api/v1/brain/decide", json=decide_data)
        self.assertEqual(res_decide.status_code, 200)
        json_decide = res_decide.json()
        self.assertIn("Recommending action 'Trigger inventory restock request'", json_decide["decision"])
        self.assertEqual(len(json_decide["similar_episodes"]), 1)
        self.assertEqual(json_decide["similar_episodes"][0]["id"], inserted_id)
        self.assertLess(json_decide["similar_episodes"][0]["distance"], 1e-5)

        # 3. Test working memory endpoint
        res_set = self.client.post("/api/v1/brain/working", json={"key": "user_id", "value": "usr-999"})
        self.assertEqual(res_set.status_code, 200)

        res_get = self.client.get("/api/v1/brain/working/user_id")
        self.assertEqual(res_get.status_code, 200)
        self.assertEqual(res_get.json()["value"], "usr-999")

        res_del = self.client.delete("/api/v1/brain/working/user_id")
        self.assertEqual(res_del.status_code, 200)

        res_get_null = self.client.get("/api/v1/brain/working/user_id")
        self.assertEqual(res_get_null.status_code, 200)
        self.assertIsNone(res_get_null.json()["value"])

if __name__ == "__main__":
    unittest.main()
