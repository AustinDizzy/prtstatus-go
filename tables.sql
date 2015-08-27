--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: updates; Type: TABLE; Schema: public; Owner: prtstatus; Tablespace: 
--

CREATE TABLE updates (
    id integer NOT NULL,
    status integer NOT NULL,
    message text NOT NULL,
    "timestamp" bigint NOT NULL,
    stations text[],
    busses_dispatched boolean
);


ALTER TABLE public.updates OWNER TO prtstatus;

--
-- Name: updates_id_seq; Type: SEQUENCE; Schema: public; Owner: prtstatus
--

CREATE SEQUENCE updates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.updates_id_seq OWNER TO prtstatus;

--
-- Name: updates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: prtstatus
--

ALTER SEQUENCE updates_id_seq OWNED BY updates.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres; Tablespace: 
--

CREATE TABLE users (
    registration_id text NOT NULL,
    tokens bytea,
    device text,
    registration_date timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT users_device_check CHECK ((device = ANY (ARRAY['android'::text, 'glass'::text, 'ios'::text])))
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: id; Type: DEFAULT; Schema: public; Owner: prtstatus
--

ALTER TABLE ONLY updates ALTER COLUMN id SET DEFAULT nextval('updates_id_seq'::regclass);


--
-- Name: updates_pkey; Type: CONSTRAINT; Schema: public; Owner: prtstatus; Tablespace: 
--

ALTER TABLE ONLY updates
    ADD CONSTRAINT updates_pkey PRIMARY KEY (id);


--
-- Name: users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres; Tablespace: 
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (registration_id);


--
-- Name: public; Type: ACL; Schema: -; Owner: postgres
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM postgres;
GRANT ALL ON SCHEMA public TO postgres;
GRANT ALL ON SCHEMA public TO PUBLIC;


--
-- Name: users; Type: ACL; Schema: public; Owner: postgres
--

REVOKE ALL ON TABLE users FROM PUBLIC;
REVOKE ALL ON TABLE users FROM postgres;
GRANT ALL ON TABLE users TO postgres;
GRANT ALL ON TABLE users TO prtstatus;


--
-- PostgreSQL database dump complete
--

