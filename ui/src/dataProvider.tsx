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
        const { page, perPage } = params.pagination;

        // Create query with pagination params.
        const query = {
            'page[number]': page,
            'page[size]': perPage,
        };

        // Add all filter params to query.
        Object.keys(params.filter || {}).forEach((key) => {
            query[`filter[${key}]`] = params.filter[key];
        });

        // Add sort parameter
        if (params.sort && params.sort.field) {
            const prefix = params.sort.order === 'ASC' ? '' : '-';
            query["sort"] = `${prefix}${params.sort.field}`;
        }

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
                        total: json.meta["count"],
                    }
                )
            }
        );
    },

    getOne: (resource, params) =>
        httpClient(`${apiUrl}/${resource}/${params.id}`).then(({ json }) => ({
            data: json.data,
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
            body: JSON.stringify({ data: { id: id, attributes: { inputs: [ {cid: params.data.cid}]} } }),
        }).then(({ json }) => ({
            data: json.data,
        }))
    },

    delete: (resource, params) => {
        return Promise.reject(new HttpError("API Error", 500, "delete not implemented"))
    },

    deleteMany: (resource, params) => {
        return Promise.reject(new HttpError("API Error", 500, "deleteMany not implemented"))
    }
};