import axios from "axios"
import type { AxiosResponse } from "axios"

const BASE_URL = "http://localhost:8000"

const ping: () => Promise<AxiosResponse> = function () {
  const url = new URL(BASE_URL)
  return axios.get(url.toString())
}

export { ping }
