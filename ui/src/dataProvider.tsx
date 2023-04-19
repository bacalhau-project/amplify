import { fetchUtils } from "react-admin";
import { stringify } from "query-string";
import { v4 as uuidv4 } from 'uuid';
import { HttpError } from 'react-admin'

const apiUrl = import.meta.env.VITE_AMPLIFY_API || '/api/v0';
console.log("Amplify API: ", apiUrl);
const httpClient = fetchUtils.fetchJson;

// TypeScript users must reference the type `DataProvider`
export const dataProvider = {
    getList: (resource, params) => {
        const query = {
        };
        const url = `${apiUrl}/${resource}?${stringify(query)}`;
        const options = {
            method: 'GET',
            headers: new Headers({ Accept: 'application/json' }),
        };

        return httpClient(url, options).then(
            ({ headers, json }) => {
                return (
                    {
                        data: json.data,
                        total: 10,
                    }
                )
            }
        );
    },

    getOne: (resource, params) =>
        httpClient(`${apiUrl}/${resource}/${params.id}`).then(({ json }) => ({
            data: json,
        })),

    getMany: (resource, params) => {
        return Promise.reject(new HttpError("API Error", 500, "getMany not implemented"))
    },

    getManyReference: (resource, params) => {
        return Promise.reject(new HttpError("API Error", 500, "getManyReference not implemented"))
    },

    update: (resource, params) => {
        return Promise.reject(new HttpError("API Error", 500, "update not implemented"))
    },

    updateMany: (resource, params) => {
        return Promise.reject(new HttpError("API Error", 500, "updateMany not implemented"))
    },

    create: (resource, params) => {
        var id = uuidv4()
        return httpClient(`${apiUrl}/${resource}/${id}`, {
            method: 'PUT',
            body: JSON.stringify(params.data),
        }).then(({ json }) => ({
            data: { ...params.data, id: id },
        }))
    },

    delete: (resource, params) => {
        return Promise.reject(new HttpError("API Error", 500, "delete not implemented"))
    },

    deleteMany: (resource, params) => {
        return Promise.reject(new HttpError("API Error", 500, "deleteMany not implemented"))
    }
};