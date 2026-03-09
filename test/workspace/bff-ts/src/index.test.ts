import request from "supertest";
import { app } from "./index";

process.env.NODE_ENV = "test";

describe("GET /api/items", () => {
  it("returns item list", async () => {
    const res = await request(app).get("/api/items");
    expect(res.status).toBe(200);
    expect(res.body).toEqual([{ id: 1, name: "Item 1" }]);
  });
});
