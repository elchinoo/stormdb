--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5 (EDB Postgres Extended Server 17.5.0)
-- Dumped by pg_dump version 17.5

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: movies_json; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_json (
    ai_myid integer NOT NULL,
    imdb_id character varying(255),
    json_column json NOT NULL
);


ALTER TABLE public.movies_json OWNER TO edb_admin;

--
-- Name: movies_json_ai_myid_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_json_ai_myid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_json_ai_myid_seq OWNER TO edb_admin;

--
-- Name: movies_json_ai_myid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_json_ai_myid_seq OWNED BY public.movies_json.ai_myid;


--
-- Name: movies_json_generated; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_json_generated (
    ai_myid integer NOT NULL,
    imdb_id character varying(255) GENERATED ALWAYS AS ((json_column ->> 'imdb_id'::text)) STORED,
    title character varying(255) GENERATED ALWAYS AS ((json_column ->> 'title'::text)) STORED,
    imdb_rating numeric(5,2) GENERATED ALWAYS AS (((json_column ->> 'imdb_rating'::text))::numeric) STORED,
    overview text GENERATED ALWAYS AS ((json_column ->> 'overview'::text)) STORED,
    director jsonb GENERATED ALWAYS AS (((json_column ->> 'director'::text))::json) STORED,
    country character varying(100) GENERATED ALWAYS AS ((json_column ->> 'country'::text)) STORED,
    jsonb_column jsonb,
    json_column json,
    cast_test jsonb GENERATED ALWAYS AS (((json_column ->> 'cast'::text))::jsonb) STORED,
    cast_is_null boolean GENERATED ALWAYS AS (
CASE
    WHEN ((jsonb_column ->> 'cast'::text) IS NULL) THEN true
    ELSE false
END) STORED
);


ALTER TABLE public.movies_json_generated OWNER TO edb_admin;

--
-- Name: movies_json_generated_ai_myid_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_json_generated_ai_myid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_json_generated_ai_myid_seq OWNER TO edb_admin;

--
-- Name: movies_json_generated_ai_myid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_json_generated_ai_myid_seq OWNED BY public.movies_json_generated.ai_myid;


--
-- Name: movies_json_generated_crash; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_json_generated_crash (
    ai_myid integer NOT NULL,
    imdb_id character varying(255) GENERATED ALWAYS AS ((json_column ->> 'imdb_id'::text)) STORED,
    title character varying(255) GENERATED ALWAYS AS ((json_column ->> 'title'::text)) STORED,
    imdb_rating numeric(5,2) GENERATED ALWAYS AS (((json_column ->> 'imdb_rating'::text))::numeric) STORED,
    overview text GENERATED ALWAYS AS ((json_column ->> 'overview'::text)) STORED,
    director jsonb GENERATED ALWAYS AS (((json_column ->> 'director'::text))::json) STORED,
    country character varying(100) GENERATED ALWAYS AS ((json_column ->> 'country'::text)) STORED,
    jsonb_column jsonb,
    json_column json
);


ALTER TABLE public.movies_json_generated_crash OWNER TO edb_admin;

--
-- Name: movies_json_generated_crash_ai_myid_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_json_generated_crash_ai_myid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_json_generated_crash_ai_myid_seq OWNER TO edb_admin;

--
-- Name: movies_json_generated_crash_ai_myid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_json_generated_crash_ai_myid_seq OWNED BY public.movies_json_generated_crash.ai_myid;


--
-- Name: movies_jsonb; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_jsonb (
    ai_myid integer NOT NULL,
    imdb_id character varying(255),
    jsonb_column jsonb NOT NULL,
    title character varying(255) GENERATED ALWAYS AS ((jsonb_column ->> 'title'::text)) STORED
);


ALTER TABLE public.movies_jsonb OWNER TO edb_admin;

--
-- Name: movies_jsonb_ai_myid_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_jsonb_ai_myid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_jsonb_ai_myid_seq OWNER TO edb_admin;

--
-- Name: movies_jsonb_ai_myid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_jsonb_ai_myid_seq OWNED BY public.movies_jsonb.ai_myid;


--
-- Name: movies_jsonb_generated_1; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_jsonb_generated_1 (
    ai_myid integer NOT NULL,
    imdb_id character varying(255) GENERATED ALWAYS AS ((jsonb_column ->> 'imdb_id'::text)) STORED,
    jsonb_column jsonb,
    json_column json
);


ALTER TABLE public.movies_jsonb_generated_1 OWNER TO edb_admin;

--
-- Name: movies_jsonb_generated_1_ai_myid_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_jsonb_generated_1_ai_myid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_jsonb_generated_1_ai_myid_seq OWNER TO edb_admin;

--
-- Name: movies_jsonb_generated_1_ai_myid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_jsonb_generated_1_ai_myid_seq OWNED BY public.movies_jsonb_generated_1.ai_myid;


--
-- Name: movies_jsonb_generated_2; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_jsonb_generated_2 (
    ai_myid integer NOT NULL,
    imdb_id character varying(255) GENERATED ALWAYS AS ((jsonb_column ->> 'imdb_id'::text)) STORED,
    title character varying(255) GENERATED ALWAYS AS ((jsonb_column ->> 'title'::text)) STORED,
    imdb_rating numeric(5,2) GENERATED ALWAYS AS (((jsonb_column ->> 'imdb_rating'::text))::numeric) STORED,
    overview text GENERATED ALWAYS AS ((jsonb_column ->> 'overview'::text)) STORED,
    director jsonb GENERATED ALWAYS AS (((jsonb_column ->> 'director'::text))::json) STORED,
    country character varying(100) GENERATED ALWAYS AS ((jsonb_column ->> 'country'::text)) STORED,
    jsonb_column jsonb,
    json_column json
);


ALTER TABLE public.movies_jsonb_generated_2 OWNER TO edb_admin;

--
-- Name: movies_jsonb_generated_2_ai_myid_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_jsonb_generated_2_ai_myid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_jsonb_generated_2_ai_myid_seq OWNER TO edb_admin;

--
-- Name: movies_jsonb_generated_2_ai_myid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_jsonb_generated_2_ai_myid_seq OWNED BY public.movies_jsonb_generated_2.ai_myid;


--
-- Name: movies_normalized_actors; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_normalized_actors (
    ai_actor_id integer NOT NULL,
    actor_id character varying(50),
    actor_name character varying(500)
);


ALTER TABLE public.movies_normalized_actors OWNER TO edb_admin;

--
-- Name: movies_normalized_actors_ai_actor_id_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_normalized_actors_ai_actor_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_normalized_actors_ai_actor_id_seq OWNER TO edb_admin;

--
-- Name: movies_normalized_actors_ai_actor_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_normalized_actors_ai_actor_id_seq OWNED BY public.movies_normalized_actors.ai_actor_id;


--
-- Name: movies_normalized_aggregate_ratings; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_normalized_aggregate_ratings (
    ai_myid integer NOT NULL,
    user_rating integer,
    up_votes integer,
    down_votes integer,
    imdb_rating integer
);


ALTER TABLE public.movies_normalized_aggregate_ratings OWNER TO edb_admin;

--
-- Name: movies_normalized_aggregate_ratings_ai_myid_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_normalized_aggregate_ratings_ai_myid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_normalized_aggregate_ratings_ai_myid_seq OWNER TO edb_admin;

--
-- Name: movies_normalized_aggregate_ratings_ai_myid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_normalized_aggregate_ratings_ai_myid_seq OWNED BY public.movies_normalized_aggregate_ratings.ai_myid;


--
-- Name: movies_normalized_cast; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_normalized_cast (
    inc_id integer NOT NULL,
    ai_actor_id integer,
    ai_myid integer,
    actor_character character varying(500)
);


ALTER TABLE public.movies_normalized_cast OWNER TO edb_admin;

--
-- Name: movies_normalized_cast_inc_id_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_normalized_cast_inc_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_normalized_cast_inc_id_seq OWNER TO edb_admin;

--
-- Name: movies_normalized_cast_inc_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_normalized_cast_inc_id_seq OWNED BY public.movies_normalized_cast.inc_id;


--
-- Name: movies_normalized_director; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_normalized_director (
    director_id integer NOT NULL,
    ai_myid integer,
    director character varying(500)
);


ALTER TABLE public.movies_normalized_director OWNER TO edb_admin;

--
-- Name: movies_normalized_director_director_id_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_normalized_director_director_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_normalized_director_director_id_seq OWNER TO edb_admin;

--
-- Name: movies_normalized_director_director_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_normalized_director_director_id_seq OWNED BY public.movies_normalized_director.director_id;


--
-- Name: movies_normalized_genres; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_normalized_genres (
    genre_id integer NOT NULL,
    ai_myid integer NOT NULL
);


ALTER TABLE public.movies_normalized_genres OWNER TO edb_admin;

--
-- Name: movies_normalized_genres_tags; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_normalized_genres_tags (
    genre_id integer NOT NULL,
    genre character varying(500)
);


ALTER TABLE public.movies_normalized_genres_tags OWNER TO edb_admin;

--
-- Name: movies_normalized_genres_tags_genre_id_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_normalized_genres_tags_genre_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_normalized_genres_tags_genre_id_seq OWNER TO edb_admin;

--
-- Name: movies_normalized_genres_tags_genre_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_normalized_genres_tags_genre_id_seq OWNED BY public.movies_normalized_genres_tags.genre_id;


--
-- Name: movies_normalized_meta; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_normalized_meta (
    ai_myid integer NOT NULL,
    imdb_id character varying(32),
    title character varying(255),
    imdb_rating numeric(5,2),
    year integer,
    country character varying(100),
    overview text,
    json_column jsonb,
    upvotes integer,
    downvotes integer
);


ALTER TABLE public.movies_normalized_meta OWNER TO edb_admin;

--
-- Name: movies_normalized_meta_ai_myid_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_normalized_meta_ai_myid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_normalized_meta_ai_myid_seq OWNER TO edb_admin;

--
-- Name: movies_normalized_meta_ai_myid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_normalized_meta_ai_myid_seq OWNED BY public.movies_normalized_meta.ai_myid;


--
-- Name: movies_normalized_user_comments; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_normalized_user_comments (
    comment_id integer NOT NULL,
    ai_myid integer,
    rating integer,
    comment text,
    imdb_id character varying(20),
    comment_add_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    comment_update_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.movies_normalized_user_comments OWNER TO edb_admin;

--
-- Name: movies_normalized_user_comments_comment_id_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_normalized_user_comments_comment_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_normalized_user_comments_comment_id_seq OWNER TO edb_admin;

--
-- Name: movies_normalized_user_comments_comment_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_normalized_user_comments_comment_id_seq OWNED BY public.movies_normalized_user_comments.comment_id;


--
-- Name: movies_viewed_logs; Type: TABLE; Schema: public; Owner: edb_admin
--

CREATE TABLE public.movies_viewed_logs (
    view_id integer NOT NULL,
    ai_myid integer,
    imdb_id character varying(32),
    watched_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    watched_user_id integer,
    time_watched_sec integer,
    encoded_data character varying(500),
    json_payload jsonb,
    json_imdb_id character varying(255) GENERATED ALWAYS AS ((json_payload ->> 'imdb_id'::text)) STORED
);


ALTER TABLE public.movies_viewed_logs OWNER TO edb_admin;

--
-- Name: movies_viewed_logs_view_id_seq; Type: SEQUENCE; Schema: public; Owner: edb_admin
--

CREATE SEQUENCE public.movies_viewed_logs_view_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_viewed_logs_view_id_seq OWNER TO edb_admin;

--
-- Name: movies_viewed_logs_view_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: edb_admin
--

ALTER SEQUENCE public.movies_viewed_logs_view_id_seq OWNED BY public.movies_viewed_logs.view_id;


--
-- Name: voting_count_history; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.voting_count_history (
    ai_myid integer NOT NULL,
    store_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    title character varying(255) NOT NULL,
    imdb_id character varying(20),
    comment_count bigint DEFAULT '0'::bigint NOT NULL,
    max_rate integer,
    avg_rate numeric(14,4) DEFAULT NULL::numeric,
    upvotes numeric(32,0) DEFAULT NULL::numeric,
    downvotes numeric(32,0) DEFAULT NULL::numeric
);


ALTER TABLE public.voting_count_history OWNER TO postgres;

--
-- Name: movies_json ai_myid; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_json ALTER COLUMN ai_myid SET DEFAULT nextval('public.movies_json_ai_myid_seq'::regclass);


--
-- Name: movies_json_generated ai_myid; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_json_generated ALTER COLUMN ai_myid SET DEFAULT nextval('public.movies_json_generated_ai_myid_seq'::regclass);


--
-- Name: movies_json_generated_crash ai_myid; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_json_generated_crash ALTER COLUMN ai_myid SET DEFAULT nextval('public.movies_json_generated_crash_ai_myid_seq'::regclass);


--
-- Name: movies_jsonb ai_myid; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_jsonb ALTER COLUMN ai_myid SET DEFAULT nextval('public.movies_jsonb_ai_myid_seq'::regclass);


--
-- Name: movies_jsonb_generated_1 ai_myid; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_jsonb_generated_1 ALTER COLUMN ai_myid SET DEFAULT nextval('public.movies_jsonb_generated_1_ai_myid_seq'::regclass);


--
-- Name: movies_jsonb_generated_2 ai_myid; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_jsonb_generated_2 ALTER COLUMN ai_myid SET DEFAULT nextval('public.movies_jsonb_generated_2_ai_myid_seq'::regclass);


--
-- Name: movies_normalized_actors ai_actor_id; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_actors ALTER COLUMN ai_actor_id SET DEFAULT nextval('public.movies_normalized_actors_ai_actor_id_seq'::regclass);


--
-- Name: movies_normalized_aggregate_ratings ai_myid; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_aggregate_ratings ALTER COLUMN ai_myid SET DEFAULT nextval('public.movies_normalized_aggregate_ratings_ai_myid_seq'::regclass);


--
-- Name: movies_normalized_cast inc_id; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_cast ALTER COLUMN inc_id SET DEFAULT nextval('public.movies_normalized_cast_inc_id_seq'::regclass);


--
-- Name: movies_normalized_director director_id; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_director ALTER COLUMN director_id SET DEFAULT nextval('public.movies_normalized_director_director_id_seq'::regclass);


--
-- Name: movies_normalized_genres_tags genre_id; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_genres_tags ALTER COLUMN genre_id SET DEFAULT nextval('public.movies_normalized_genres_tags_genre_id_seq'::regclass);


--
-- Name: movies_normalized_meta ai_myid; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_meta ALTER COLUMN ai_myid SET DEFAULT nextval('public.movies_normalized_meta_ai_myid_seq'::regclass);


--
-- Name: movies_normalized_user_comments comment_id; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_user_comments ALTER COLUMN comment_id SET DEFAULT nextval('public.movies_normalized_user_comments_comment_id_seq'::regclass);


--
-- Name: movies_viewed_logs view_id; Type: DEFAULT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_viewed_logs ALTER COLUMN view_id SET DEFAULT nextval('public.movies_viewed_logs_view_id_seq'::regclass);


--
-- Name: movies_json_generated_crash movies_json_generated_crash_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_json_generated_crash
    ADD CONSTRAINT movies_json_generated_crash_pkey PRIMARY KEY (ai_myid);


--
-- Name: movies_json_generated movies_json_generated_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_json_generated
    ADD CONSTRAINT movies_json_generated_pkey PRIMARY KEY (ai_myid);


--
-- Name: movies_json movies_json_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_json
    ADD CONSTRAINT movies_json_pkey PRIMARY KEY (ai_myid);


--
-- Name: movies_jsonb_generated_1 movies_jsonb_generated_1_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_jsonb_generated_1
    ADD CONSTRAINT movies_jsonb_generated_1_pkey PRIMARY KEY (ai_myid);


--
-- Name: movies_jsonb_generated_2 movies_jsonb_generated_2_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_jsonb_generated_2
    ADD CONSTRAINT movies_jsonb_generated_2_pkey PRIMARY KEY (ai_myid);


--
-- Name: movies_jsonb movies_jsonb_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_jsonb
    ADD CONSTRAINT movies_jsonb_pkey PRIMARY KEY (ai_myid);


--
-- Name: movies_normalized_actors movies_normalized_actors_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_actors
    ADD CONSTRAINT movies_normalized_actors_pkey PRIMARY KEY (ai_actor_id);


--
-- Name: movies_normalized_aggregate_ratings movies_normalized_aggregate_ratings_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_aggregate_ratings
    ADD CONSTRAINT movies_normalized_aggregate_ratings_pkey PRIMARY KEY (ai_myid);


--
-- Name: movies_normalized_cast movies_normalized_cast_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_cast
    ADD CONSTRAINT movies_normalized_cast_pkey PRIMARY KEY (inc_id);


--
-- Name: movies_normalized_director movies_normalized_director_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_director
    ADD CONSTRAINT movies_normalized_director_pkey PRIMARY KEY (director_id);


--
-- Name: movies_normalized_genres movies_normalized_genres_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_genres
    ADD CONSTRAINT movies_normalized_genres_pkey PRIMARY KEY (genre_id, ai_myid);


--
-- Name: movies_normalized_genres_tags movies_normalized_genres_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_genres_tags
    ADD CONSTRAINT movies_normalized_genres_tags_pkey PRIMARY KEY (genre_id);


--
-- Name: movies_normalized_meta movies_normalized_meta_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_meta
    ADD CONSTRAINT movies_normalized_meta_pkey PRIMARY KEY (ai_myid);


--
-- Name: movies_normalized_user_comments movies_normalized_user_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_normalized_user_comments
    ADD CONSTRAINT movies_normalized_user_comments_pkey PRIMARY KEY (comment_id);


--
-- Name: movies_viewed_logs movies_viewed_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: edb_admin
--

ALTER TABLE ONLY public.movies_viewed_logs
    ADD CONSTRAINT movies_viewed_logs_pkey PRIMARY KEY (view_id);


--
-- Name: voting_count_history voting_count_history_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.voting_count_history
    ADD CONSTRAINT voting_count_history_pkey PRIMARY KEY (title, ai_myid, store_time);


--
-- Name: actor_id_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE UNIQUE INDEX actor_id_idx ON public.movies_normalized_actors USING btree (actor_id);


--
-- Name: cnt_hist_ai_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX cnt_hist_ai_idx ON public.voting_count_history USING btree (ai_myid);


--
-- Name: crash_gen_func_title_index; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX crash_gen_func_title_index ON public.movies_json_generated_crash USING btree ((((json_column ->> 'title'::text))::character varying));


--
-- Name: crash_gen_gin_index; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX crash_gen_gin_index ON public.movies_json_generated_crash USING gin (jsonb_column);


--
-- Name: crash_gen_imdb_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE UNIQUE INDEX crash_gen_imdb_idx ON public.movies_json_generated_crash USING btree (imdb_id);


--
-- Name: crash_gen_title_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX crash_gen_title_idx ON public.movies_json_generated_crash USING btree (title);


--
-- Name: director_mv_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX director_mv_idx ON public.movies_normalized_director USING btree (ai_myid);


--
-- Name: gen_func_index_cast; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX gen_func_index_cast ON public.movies_json_generated USING btree ((
CASE
    WHEN ((jsonb_column ->> 'cast'::text) = 'null'::text) THEN 0
    ELSE 1
END));


--
-- Name: gen_func_title_index; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX gen_func_title_index ON public.movies_json_generated USING btree ((((json_column ->> 'title'::text))::character varying));


--
-- Name: gen_gin_index; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX gen_gin_index ON public.movies_json_generated USING gin (jsonb_column);


--
-- Name: gen_imdb_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE UNIQUE INDEX gen_imdb_idx ON public.movies_json_generated USING btree (imdb_id);


--
-- Name: gen_title_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX gen_title_idx ON public.movies_json_generated USING btree (title);


--
-- Name: genre_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE UNIQUE INDEX genre_idx ON public.movies_normalized_genres_tags USING btree (genre);


--
-- Name: gin_index; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX gin_index ON public.movies_normalized_meta USING gin (json_column);


--
-- Name: idx_cast_is_null; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX idx_cast_is_null ON public.movies_json_generated USING btree (cast_is_null);


--
-- Name: idx_comments_com_time; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX idx_comments_com_time ON public.movies_normalized_user_comments USING btree (comment_add_time);


--
-- Name: idx_comments_id; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX idx_comments_id ON public.movies_normalized_user_comments USING btree (ai_myid, comment_add_time);


--
-- Name: idx_nc_char; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX idx_nc_char ON public.movies_normalized_cast USING btree (actor_character);


--
-- Name: idx_nc_id; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX idx_nc_id ON public.movies_normalized_cast USING btree (ai_myid);


--
-- Name: idx_nc_id2; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX idx_nc_id2 ON public.movies_normalized_cast USING btree (ai_actor_id, ai_myid);


--
-- Name: idx_nmm_country_year; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX idx_nmm_country_year ON public.movies_normalized_meta USING btree (country, year, imdb_rating);


--
-- Name: idx_nmm_rate; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX idx_nmm_rate ON public.movies_normalized_meta USING btree (imdb_rating);


--
-- Name: idx_nmm_title; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX idx_nmm_title ON public.movies_normalized_meta USING btree (title);


--
-- Name: imdb_id_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE UNIQUE INDEX imdb_id_idx ON public.movies_normalized_meta USING btree (imdb_id);


--
-- Name: m_view_idx1; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX m_view_idx1 ON public.movies_viewed_logs USING btree (ai_myid, watched_time);


--
-- Name: m_view_idx2; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX m_view_idx2 ON public.movies_viewed_logs USING btree (watched_time);


--
-- Name: m_view_idx3; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE INDEX m_view_idx3 ON public.movies_viewed_logs USING btree (watched_user_id);


--
-- Name: movies_json_imdb_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE UNIQUE INDEX movies_json_imdb_idx ON public.movies_json USING btree (imdb_id);


--
-- Name: movies_jsonb_imdb_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE UNIQUE INDEX movies_jsonb_imdb_idx ON public.movies_jsonb USING btree (imdb_id);


--
-- Name: u_cast_idx; Type: INDEX; Schema: public; Owner: edb_admin
--

CREATE UNIQUE INDEX u_cast_idx ON public.movies_normalized_cast USING btree (ai_myid, ai_actor_id, actor_character);


--
-- PostgreSQL database dump complete
--

