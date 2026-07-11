import type { FetchListPageMessage, WSIncomingMessage } from "./protocol.js";

const request: FetchListPageMessage = {
  type: "fetch_list_page",
  id: "list-1",
  payload: { url: "https://www.ozon.ru/search/?text=storage" },
};

const incoming: WSIncomingMessage = request;
void incoming;
