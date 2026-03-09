import express, { Request, Response } from "express";

const app = express();
const API_GO_BASE = process.env.API_GO_URL ?? "http://localhost:8080";

// In test/standalone mode, return a mock response.
// In production, this would proxy to api-go.
app.get("/api/items", async (_req: Request, res: Response) => {
  if (process.env.NODE_ENV === "test") {
    res.json([{ id: 1, name: "Item 1" }]);
    return;
  }
  const upstream = await fetch(`${API_GO_BASE}/api/items`);
  const body = await upstream.json();
  res.status(upstream.status).json(body);
});

export { app };

if (require.main === module) {
  app.listen(3000, () => console.log("BFF listening on :3000"));
}
